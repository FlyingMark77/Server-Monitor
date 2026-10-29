package monitor

import (
	"fmt"
	"sort"
	"strings"
)

// Only fresh, complete samples can establish availability. Zero memory means
// exactly zero MiB as reported by nvidia-smi; the utilization boundary is strict.
func idleGPUIndices(row ServerSnapshot) []int {
	indices := []int{}
	if !row.Server.Enabled || row.Status != "live" {
		return indices
	}
	for _, gpu := range row.GPUs {
		if gpu.MemUsed != nil && gpu.Util != nil && *gpu.MemUsed == 0 && *gpu.Util >= 0 && *gpu.Util < 5 {
			indices = append(indices, gpu.Index)
		}
	}
	sort.Ints(indices)
	return indices
}

func MenuTitle(snapshot Snapshot) string {
	parts := []string{}
	for _, row := range snapshot.Servers {
		if !row.Server.Enabled {
			continue
		}
		value := "—"
		if row.Status == "live" || row.Status == "empty" {
			value = fmt.Sprintf("%d空闲", len(row.IdleGPUIndices))
		}
		parts = append(parts, row.Server.Name+" "+value)
	}
	if len(parts) == 0 {
		if len(snapshot.Servers) > 0 {
			return "已暂停"
		}
		return "未配置"
	}
	return strings.Join(parts, " · ")
}
