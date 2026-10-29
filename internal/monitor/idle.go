package monitor

import (
	"fmt"
	"sort"
	"strings"
)

// Only fresh, complete samples can establish availability. Both bounds are strict.
func idleGPUIndices(row ServerSnapshot, config Config) []int {
	memoryMiB := config.IdleMemoryGB * 1e9 / (1024 * 1024)
	indices := []int{}
	if !row.Server.Enabled || row.Status != "live" {
		return indices
	}
	for _, gpu := range row.GPUs {
		if gpu.MemUsed != nil && gpu.Util != nil && *gpu.MemUsed >= 0 && *gpu.MemUsed < memoryMiB && *gpu.Util >= 0 && *gpu.Util < config.IdleUtilPercent {
			indices = append(indices, gpu.Index)
		}
	}
	sort.Ints(indices)
	return indices
}

func MenuTitle(snapshot Snapshot) string {
	parts := []string{}
	for index, row := range snapshot.Servers {
		if !row.Server.Enabled {
			continue
		}
		value := "—"
		if row.Status == "live" || row.Status == "empty" {
			value = fmt.Sprintf("%d空闲", len(row.IdleGPUIndices))
		}
		parts = append(parts, fmt.Sprintf("#%d %s", index+1, value))
	}
	if len(parts) == 0 {
		if len(snapshot.Servers) > 0 {
			return "已暂停"
		}
		return "未配置"
	}
	return strings.Join(parts, " · ")
}
