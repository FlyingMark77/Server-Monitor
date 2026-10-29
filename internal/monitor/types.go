package monitor

import "time"

type Server struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Target  string `json:"target"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}
type Config struct {
	Version  int      `json:"version"`
	Servers  []Server `json:"servers"`
	Interval int      `json:"interval"`
	Theme    string   `json:"theme"`
}
type GPU struct {
	UUID     string   `json:"uuid"`
	Index    int      `json:"index"`
	Name     string   `json:"name"`
	Util     *float64 `json:"util"`
	MemUsed  *float64 `json:"memUsed"`
	MemTotal *float64 `json:"memTotal"`
	Temp     *float64 `json:"temp"`
}
type ServerSnapshot struct {
	Server         Server `json:"server"`
	Status         string `json:"status"`
	GPUs           []GPU  `json:"gpus"`
	IdleGPUIndices []int  `json:"idleGPUIndices"`
	LastSuccess    int64  `json:"lastSuccess"`
	NextRetry      int64  `json:"nextRetry"`
	ErrorCode      string `json:"errorCode"`
	ErrorMessage   string `json:"errorMessage"`
}
type Snapshot struct {
	Revision  uint64           `json:"revision"`
	Timestamp int64            `json:"timestamp"`
	Servers   []ServerSnapshot `json:"servers"`
	Online    int              `json:"online"`
	Enabled   int              `json:"enabled"`
	GPUCount  int              `json:"gpuCount"`
}

func DefaultConfig() Config {
	return Config{Version: 1, Servers: []Server{}, Interval: 2, Theme: "system"}
}
func staleAfter(interval int) time.Duration { return time.Duration(max(10, interval*3)) * time.Second }
