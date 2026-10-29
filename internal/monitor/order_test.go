package monitor

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestReorderPersistsWithoutRestartingCollectors(t *testing.T) {
	var created atomic.Int32
	e := NewEngine(func(server Server) Sampler {
		created.Add(1)
		return mockSampler{q: func(context.Context) ([]GPU, error) {
			return []GPU{{UUID: "GPU-" + server.ID}}, nil
		}}
	}, nil)
	defer e.Close()
	c := twoConfig()
	e.Apply(c)
	eventually(t, func() bool { return e.Snapshot().Online == 2 })
	c.Servers[0], c.Servers[1] = c.Servers[1], c.Servers[0]
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	restored, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	e.Apply(restored)
	s := e.Snapshot()
	if s.Servers[0].Server.ID != "b" || s.Servers[0].GPUs[0].UUID != "GPU-b" || s.Servers[1].Server.ID != "a" || s.Servers[1].GPUs[0].UUID != "GPU-a" {
		t.Fatalf("order or GPU identity changed: %+v", s)
	}
	if created.Load() != 2 {
		t.Fatal("reorder restarted collectors")
	}
}
