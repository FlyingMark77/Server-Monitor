// monitor-check exercises the same collector and scheduler as the desktop app.
// Host aliases and its output are private local verification data, not fixtures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"servermonitor/internal/monitor"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	hosts := flag.String("hosts", "", "comma-separated SSH aliases")
	duration := flag.Duration("duration", time.Minute, "verification duration")
	interval := flag.Int("interval", 2, "sample interval in seconds")
	flag.Parse()
	if *hosts == "" {
		fmt.Fprintln(os.Stderr, "provide -hosts")
		os.Exit(2)
	}
	c := monitor.DefaultConfig()
	c.Interval = *interval
	for i, h := range strings.Split(*hosts, ",") {
		c.Servers = append(c.Servers, monitor.Server{ID: fmt.Sprint(i), Name: fmt.Sprintf("server-%d", i+1), Target: strings.TrimSpace(h), Enabled: true})
	}
	c, err := monitor.NormalizeConfig(c)
	if err != nil {
		panic(err)
	}
	client, err := monitor.NewSSHClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	type Counts struct {
		Samples     int   `json:"samples"`
		Failures    int   `json:"failureEvents"`
		GPUs        int   `json:"gpus"`
		LastSuccess int64 `json:"lastSuccess"`
	}
	counts := map[string]*Counts{}
	lastErr := map[string]int64{}
	var mu sync.Mutex
	e := monitor.NewEngine(client.NewSession, func(s monitor.Snapshot) {
		mu.Lock()
		defer mu.Unlock()
		for _, r := range s.Servers {
			v := counts[r.Server.Name]
			if v == nil {
				v = &Counts{}
				counts[r.Server.Name] = v
			}
			if r.LastSuccess > v.LastSuccess {
				v.Samples++
				v.LastSuccess = r.LastSuccess
				v.GPUs = len(r.GPUs)
			}
			if r.NextRetry > 0 && r.NextRetry != lastErr[r.Server.ID] {
				v.Failures++
				lastErr[r.Server.ID] = r.NextRetry
			}
		}
	})
	e.Apply(c)
	start := time.Now()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	timer := time.NewTimer(*duration)
	defer timer.Stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	print := func(final bool) {
		mu.Lock()
		defer mu.Unlock()
		b, _ := json.Marshal(map[string]any{"final": final, "elapsedSeconds": int(time.Since(start).Seconds()), "servers": counts})
		fmt.Println(string(b))
	}
	for {
		select {
		case <-ticker.C:
			print(false)
		case <-timer.C:
			e.Close()
			print(true)
			for _, v := range counts {
				if v.Samples == 0 {
					os.Exit(1)
				}
			}
			return
		case <-sig:
			e.Close()
			print(true)
			return
		}
	}
}
