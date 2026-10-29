package monitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGPUCSV(t *testing.T) {
	raw := "GPU-abc, 0, \"NVIDIA, Test\", 12.5, 1024, 24576, 62\nGPU-def, 1, NVIDIA Test, N/A, [Not Supported], 24576, N/A\n"
	g, err := ParseGPUs(raw)
	if err != nil || len(g) != 2 {
		t.Fatalf("%v %v", g, err)
	}
	if g[0].Name != "NVIDIA, Test" || *g[0].Util != 12.5 || g[1].Util != nil || g[1].MemUsed != nil || g[1].Temp != nil {
		t.Fatalf("bad values: %+v", g)
	}
	for _, bad := range []string{"banner\n" + raw, "GPU-a, 0, Test, NaN, 1, 2, 3", "GPU-a, 0, Test, 101, 1, 2, 3", raw + raw} {
		if _, err := ParseGPUs(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	g, err = ParseGPUs("")
	if err != nil || len(g) != 0 {
		t.Fatal("empty should be no GPUs")
	}
}
func TestSSHDiscoveryPreservesAlias(t *testing.T) {
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "conf.d"), 0700)
	os.WriteFile(filepath.Join(d, "config"), []byte("Host *\n IdentityFile ~/.ssh/key\nInclude conf.d/*\nHost gpu-a gpu-b\n HostName 192.0.2.1\n User demo\n Port 2222\n ProxyJump gateway\nHost *.wild !excluded\n"), 0600)
	os.WriteFile(filepath.Join(d, "conf.d", "gpu"), []byte("Include config\nHost=\"gpu-c\" # comment\n HostName 192.0.2.2\n"), 0600)
	list, err := DiscoverSSHConfig(filepath.Join(d, "config"))
	if err != nil || len(list) != 3 {
		t.Fatalf("%v %v", list, err)
	}
	for i, want := range []string{"gpu-a", "gpu-b", "gpu-c"} {
		if list[i].Target != want {
			t.Fatalf("alias changed: %+v", list[i])
		}
	}
}
func TestConfigRoundTripAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "config.json")
	c := DefaultConfig()
	c.Servers = []Server{{Name: "测试", Target: "gpu-a", Enabled: true}}
	c, err := NormalizeConfig(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(path)
	if err != nil || got.Servers[0].ID != c.Servers[0].ID {
		t.Fatalf("%v %+v", err, got)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions %v", info.Mode())
	}
	for _, s := range []string{"-oProxyCommand=bad", "host\nnext", "user@host;cmd", "host name", "$(bad)"} {
		if ValidateTarget(s) == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	c.Servers = append(c.Servers, c.Servers[0])
	if _, err = NormalizeConfig(c); err == nil {
		t.Fatal("accepted duplicate")
	}
	os.WriteFile(path, []byte("broken"), 0600)
	if _, err = LoadConfig(path); err == nil {
		t.Fatal("accepted corruption")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "broken" {
		t.Fatal("overwrote config")
	}
}

type mockSampler struct {
	q     func(context.Context) ([]GPU, error)
	close func()
}

func (m mockSampler) Query(c context.Context) ([]GPU, error) { return m.q(c) }
func (m mockSampler) Close() {
	if m.close != nil {
		m.close()
	}
}
func eventually(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}
func twoConfig() Config {
	c := DefaultConfig()
	c.Servers = []Server{{ID: "a", Name: "A", Target: "gpu-a", Enabled: true}, {ID: "b", Name: "B", Target: "gpu-b", Enabled: true}}
	return c
}
func TestIndependentServersCancellationAndLateResult(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var closed atomic.Int32
	e := NewEngine(func(s Server) Sampler {
		return mockSampler{q: func(ctx context.Context) ([]GPU, error) {
			if s.ID == "a" {
				select {
				case <-started:
				default:
					close(started)
				}
				<-release
			}
			return []GPU{{UUID: "GPU-" + s.ID}}, nil
		}, close: func() { closed.Add(1) }}
	}, nil)
	defer e.Close()
	c := twoConfig()
	e.Apply(c)
	<-started
	eventually(t, func() bool { return e.Snapshot().Online == 1 })
	c.Servers = c.Servers[1:]
	e.Apply(c)
	close(release)
	eventually(t, func() bool { return closed.Load() == 1 })
	s := e.Snapshot()
	if len(s.Servers) != 1 || s.Servers[0].Server.ID != "b" || s.GPUCount != 1 {
		t.Fatalf("late result leaked: %+v", s)
	}
}
func TestRetryRetainsLastDataAndRecovers(t *testing.T) {
	var n atomic.Int32
	e := NewEngine(func(s Server) Sampler {
		return mockSampler{q: func(ctx context.Context) ([]GPU, error) {
			if n.Add(1) == 2 {
				return nil, errors.New("Permission denied")
			}
			return []GPU{{UUID: "GPU-x"}}, nil
		}}
	}, nil)
	defer e.Close()
	c := twoConfig()
	c.Servers = c.Servers[:1]
	e.Apply(c)
	eventually(t, func() bool { return e.Snapshot().Online == 1 })
	e.Retry("a")
	eventually(t, func() bool { return e.Snapshot().Servers[0].Status == "stale" })
	row := e.Snapshot().Servers[0]
	if len(row.GPUs) != 1 || row.ErrorCode != "auth_failed" || row.NextRetry <= row.LastSuccess {
		t.Fatalf("%+v", row)
	}
	e.Retry("a")
	eventually(t, func() bool { return e.Snapshot().Online == 1 })
}
func TestConcurrencyBoundAndClose(t *testing.T) {
	var active, peak, started atomic.Int32
	e := NewEngine(func(s Server) Sampler {
		return mockSampler{q: func(ctx context.Context) ([]GPU, error) {
			v := active.Add(1)
			defer active.Add(-1)
			for {
				p := peak.Load()
				if v <= p || peak.CompareAndSwap(p, v) {
					break
				}
			}
			started.Add(1)
			<-ctx.Done()
			return nil, ctx.Err()
		}}
	}, nil)
	c := DefaultConfig()
	for _, id := range []string{"a", "b", "c", "d", "e", "f"} {
		c.Servers = append(c.Servers, Server{ID: id, Name: id, Target: id, Enabled: true})
	}
	e.Apply(c)
	eventually(t, func() bool { return started.Load() == 5 })
	if peak.Load() > 5 {
		t.Fatal("unbounded parallelism")
	}
	e.Close()
	if active.Load() != 0 {
		t.Fatal("unfinished queries")
	}
}
func TestStaleSnapshotAfterSleep(t *testing.T) {
	e := NewEngine(func(s Server) Sampler {
		return mockSampler{q: func(ctx context.Context) ([]GPU, error) { <-ctx.Done(); return nil, ctx.Err() }}
	}, nil)
	defer e.Close()
	c := twoConfig()
	c.Servers = c.Servers[:1]
	e.Apply(c)
	e.mu.Lock()
	w := e.workers["a"]
	w.state.Status = "live"
	w.state.GPUs = []GPU{{UUID: "GPU-x"}}
	w.state.LastSuccess = time.Now().Add(-time.Minute).UnixMilli()
	e.mu.Unlock()
	s := e.Snapshot()
	if s.Online != 0 || s.GPUCount != 0 || s.Servers[0].Status != "stale" {
		t.Fatalf("stale totals %+v", s)
	}
}
func TestSSHArguments(t *testing.T) {
	c := &SSHClient{dir: t.TempDir(), binary: "ssh"}
	s := c.NewSession(Server{Target: "gpu-a", Port: 2222}).(*sshSession)
	args := strings.Join(s.args(), " ")
	for _, part := range []string{"BatchMode=yes", "StrictHostKeyChecking=yes", "ConnectTimeout=5", "-p 2222", "ClearAllForwardings=yes", "RemoteCommand=none"} {
		if !strings.Contains(args, part) {
			t.Errorf("missing %s", part)
		}
	}
}
