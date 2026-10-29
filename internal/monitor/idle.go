package monitor

import (
	"fmt"
	"sort"
	"strings"
)

// nvidia-smi reports MiB; the user-facing 0.5 GB threshold uses decimal GB.
const idleMemoryMiB = 500_000_000.0 / (1024 * 1024)

// Only fresh, complete samples can establish availability. Both bounds are strict.
func idleGPUIndices(row ServerSnapshot) []int {
	indices := []int{}
	if !row.Server.Enabled || row.Status != "live" {
		return indices
	}
	for _, gpu := range row.GPUs {
		if gpu.MemUsed != nil && gpu.Util != nil && *gpu.MemUsed >= 0 && *gpu.MemUsed < idleMemoryMiB && *gpu.Util >= 0 && *gpu.Util < 5 {
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
