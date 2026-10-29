package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type worker struct {
	server   Server
	cancel   context.CancelFunc
	wake     chan struct{}
	state    ServerSnapshot
	failures int
}
type Engine struct {
	mu       sync.Mutex
	emitMu   sync.Mutex
	cfg      Config
	workers  map[string]*worker
	factory  Factory
	callback func(Snapshot)
	slots    chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	revision uint64
	closed   bool
}

func NewEngine(factory Factory, callback func(Snapshot)) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{cfg: DefaultConfig(), workers: map[string]*worker{}, factory: factory, callback: callback, slots: make(chan struct{}, 5), ctx: ctx, cancel: cancel}
	e.wg.Add(1)
	go e.heartbeat()
	return e
}
func (e *Engine) Config() Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	c := e.cfg
	c.Servers = append([]Server{}, c.Servers...)
	return c
}
func (e *Engine) Apply(c Config) {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	keep := map[string]bool{}
	for _, s := range c.Servers {
		if !s.Enabled {
			continue
		}
		keep[s.ID] = true
		w := e.workers[s.ID]
		if w != nil && w.server.Target == s.Target && w.server.Port == s.Port && e.cfg.Interval == c.Interval {
			w.state.Server = s
			continue
		}
		if w != nil {
			w.cancel()
		}
		ctx, cancel := context.WithCancel(e.ctx)
		w = &worker{server: s, cancel: cancel, wake: make(chan struct{}, 1), state: ServerSnapshot{Server: s, Status: "connecting", GPUs: []GPU{}}}
		e.workers[s.ID] = w
		e.wg.Add(1)
		go e.run(ctx, w, c.Interval)
	}
	for id, w := range e.workers {
		if !keep[id] {
			w.cancel()
			delete(e.workers, id)
		}
	}
	e.cfg = c
	e.cfg.Servers = append([]Server{}, c.Servers...)
	e.mu.Unlock()
	e.emit()
}
func (e *Engine) Retry(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	w := e.workers[id]
	if w == nil {
		return fmt.Errorf("服务器未启用或已删除")
	}
	select {
	case w.wake <- struct{}{}:
	default:
	}
	return nil
}
func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked(time.Now())
}
func (e *Engine) snapshotLocked(now time.Time) Snapshot {
	s := Snapshot{Revision: e.revision, Timestamp: now.UnixMilli(), Servers: []ServerSnapshot{}}
	for _, server := range e.cfg.Servers {
		row := ServerSnapshot{Server: server, Status: "disabled", GPUs: []GPU{}}
		if server.Enabled {
			s.Enabled++
			if w := e.workers[server.ID]; w != nil {
				row = w.state
				row.Server = server
				row.GPUs = append([]GPU{}, row.GPUs...)
				if row.LastSuccess > 0 && (row.Status == "live" || row.Status == "empty") && now.Sub(time.UnixMilli(row.LastSuccess)) > staleAfter(e.cfg.Interval) {
					row.Status = "stale"
					row.ErrorMessage = "等待新数据"
				}
				if row.Status == "live" || row.Status == "empty" {
					s.Online++
					s.GPUCount += len(row.GPUs)
				}
			}
		}
		row.IdleGPUIndices = idleGPUIndices(row, e.cfg)
		s.Servers = append(s.Servers, row)
	}
	return s
}
func (e *Engine) emit() {
	e.emitMu.Lock()
	defer e.emitMu.Unlock()
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.revision++
	s := e.snapshotLocked(time.Now())
	e.mu.Unlock()
	if e.callback != nil {
		e.callback(s)
	}
}
func (e *Engine) run(ctx context.Context, w *worker, interval int) {
	defer e.wg.Done()
	sampler := e.factory(w.server)
	defer sampler.Close()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-w.wake:
		}
		if ctx.Err() != nil {
			return
		}
		select {
		case e.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		qctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		gpus, err := sampler.Query(qctx)
		cancel()
		<-e.slots
		if ctx.Err() != nil {
			return
		}
		now := time.Now()
		delay := time.Duration(interval) * time.Second
		e.mu.Lock()
		if e.workers[w.server.ID] != w {
			e.mu.Unlock()
			return
		}
		if err == nil {
			w.failures = 0
			w.state.GPUs = gpus
			w.state.LastSuccess = now.UnixMilli()
			w.state.ErrorCode = ""
			w.state.ErrorMessage = ""
			w.state.NextRetry = 0
			w.state.Status = "live"
			if len(gpus) == 0 {
				w.state.Status = "empty"
			}
		} else {
			w.failures++
			delay = retryDelay(w.failures)
			w.state.ErrorCode, w.state.ErrorMessage = ErrorInfo(err)
			w.state.NextRetry = now.Add(delay).UnixMilli()
			w.state.Status = "error"
			if w.state.LastSuccess > 0 {
				w.state.Status = "stale"
			}
		}
		e.mu.Unlock()
		e.emit()
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(delay)
	}
}
func retryDelay(failures int) time.Duration {
	delays := []int{2, 5, 10, 20, 30}
	return time.Duration(delays[min(max(0, failures-1), len(delays)-1)]) * time.Second
}
func (e *Engine) heartbeat() {
	defer e.wg.Done()
	t := time.NewTicker(time.Second)
	defer t.Stop()
	last := time.Now()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			if now.Sub(last) > 4*time.Second {
				e.mu.Lock()
				for _, w := range e.workers {
					select {
					case w.wake <- struct{}{}:
					default:
					}
				}
				e.mu.Unlock()
			}
			last = now
			e.emit()
		}
	}
}
func (e *Engine) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.closed = true
	e.cancel()
	for _, w := range e.workers {
		w.cancel()
	}
	e.mu.Unlock()
	e.wg.Wait()
}
