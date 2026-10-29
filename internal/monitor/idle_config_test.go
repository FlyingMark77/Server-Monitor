package monitor

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIdleConfigMigrationAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"servers":[],"interval":2,"theme":"system"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil || c.IdleMemoryGB != 0.5 || c.IdleUtilPercent != 5 {
		t.Fatalf("legacy config: %+v %v", c, err)
	}
	c.IdleMemoryGB, c.IdleUtilPercent = 1.25, 12
	if err := SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	c, err = LoadConfig(path)
	if err != nil || c.IdleMemoryGB != 1.25 || c.IdleUtilPercent != 12 {
		t.Fatalf("saved config: %+v %v", c, err)
	}
	for _, v := range []float64{-1, 1025, math.NaN(), math.Inf(1)} {
		bad := c
		bad.IdleMemoryGB = v
		if _, err := NormalizeConfig(bad); err == nil {
			t.Fatal("invalid memory accepted", v)
		}
	}
	for _, v := range []float64{-1, 101, math.NaN(), math.Inf(1)} {
		bad := c
		bad.IdleUtilPercent = v
		if _, err := NormalizeConfig(bad); err == nil {
			t.Fatal("invalid utilization accepted", v)
		}
	}
}

func TestIdleThresholdChangesReclassifyExistingSnapshot(t *testing.T) {
	mem, util := 800.0, 8.0
	c := DefaultConfig()
	c.Servers = []Server{{ID: "a", Enabled: true}}
	e := &Engine{cfg: c, workers: map[string]*worker{"a": {state: ServerSnapshot{Status: "live", LastSuccess: time.Now().UnixMilli(), GPUs: []GPU{{Index: 2, MemUsed: &mem, Util: &util}}}}}}
	if len(e.Snapshot().Servers[0].IdleGPUIndices) != 0 {
		t.Fatal("default threshold")
	}
	e.cfg.IdleMemoryGB = 1
	e.cfg.IdleUtilPercent = 10
	if got := e.Snapshot().Servers[0].IdleGPUIndices; len(got) != 1 || got[0] != 2 {
		t.Fatal("custom thresholds not applied", got)
	}
	e.cfg.IdleUtilPercent = 8
	if len(e.Snapshot().Servers[0].IdleGPUIndices) != 0 {
		t.Fatal("boundary must remain strict")
	}
}
