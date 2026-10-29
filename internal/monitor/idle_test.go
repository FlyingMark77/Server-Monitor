package monitor

import (
	"reflect"
	"testing"
	"time"
)

func TestIdleGPUsStrictBoundaryAndMissingData(t *testing.T) {
	n := func(v float64) *float64 { return &v }
	row := ServerSnapshot{Server: Server{Enabled: true}, Status: "live", GPUs: []GPU{
		{Index: 7, MemUsed: n(0), Util: n(4.99)},
		{Index: 2, MemUsed: n(0), Util: n(0)},
		{Index: 3, MemUsed: n(0), Util: n(5)},
		{Index: 4, MemUsed: n(1), Util: n(0)},
		{Index: 5, MemUsed: nil, Util: n(0)},
		{Index: 6, MemUsed: n(0), Util: nil},
	}}
	if got := idleGPUIndices(row); !reflect.DeepEqual(got, []int{2, 7}) {
		t.Fatalf("idle indices = %v", got)
	}
	for _, status := range []string{"stale", "error", "connecting", "disabled", "empty"} {
		row.Status = status
		if len(idleGPUIndices(row)) != 0 {
			t.Fatalf("%s exposed old availability", status)
		}
	}
	row.Status, row.Server.Enabled = "live", false
	if len(idleGPUIndices(row)) != 0 {
		t.Fatal("disabled server exposed availability")
	}
}

func TestIdleSnapshotExpiryAndMenu(t *testing.T) {
	zero := 0.0
	server := Server{ID: "a", Name: "训练机", Enabled: true}
	now := time.Now()
	e := &Engine{cfg: Config{Interval: 2, Servers: []Server{server}}, workers: map[string]*worker{
		"a": {state: ServerSnapshot{Server: server, Status: "live", LastSuccess: now.UnixMilli(), GPUs: []GPU{{Index: 3, MemUsed: &zero, Util: &zero}}}},
	}}
	fresh := e.snapshotLocked(now)
	if !reflect.DeepEqual(fresh.Servers[0].IdleGPUIndices, []int{3}) || MenuTitle(fresh) != "训练机 1空闲" {
		t.Fatalf("fresh availability: %+v / %s", fresh, MenuTitle(fresh))
	}
	stale := e.snapshotLocked(now.Add(time.Minute))
	if len(stale.Servers[0].IdleGPUIndices) != 0 || MenuTitle(stale) != "训练机 —" {
		t.Fatalf("expired availability: %+v", stale)
	}
	fresh.Servers = append(fresh.Servers, ServerSnapshot{Server: Server{Name: "推理机", Enabled: true}, Status: "empty", IdleGPUIndices: []int{}})
	if got := MenuTitle(fresh); got != "训练机 1空闲 · 推理机 0空闲" {
		t.Fatal(got)
	}
	if MenuTitle(Snapshot{}) != "未配置" || MenuTitle(Snapshot{Servers: []ServerSnapshot{{Server: Server{Enabled: false}}}}) != "已暂停" {
		t.Fatal("empty/paused menu")
	}
}
