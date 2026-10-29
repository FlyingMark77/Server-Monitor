package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"servermonitor/internal/monitor"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	miniWidth  = 460
	miniHeight = 620
	mainWidth  = 1040
	mainHeight = 760
)

type App struct {
	ctx          context.Context
	mu           sync.Mutex
	saveMu       sync.Mutex
	mode         string
	visible      bool
	engine       *monitor.Engine
	ssh          *monitor.SSHClient
	configPath   string
	startupError string
}
type InitialState struct {
	Config   monitor.Config   `json:"config"`
	Snapshot monitor.Snapshot `json:"snapshot"`
	Mode     string           `json:"mode"`
	Error    string           `json:"error"`
	Version  string           `json:"version"`
}

func NewApp() *App { return &App{mode: "main"} }
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go trayRun(a)
	path, err := monitor.ConfigPath()
	if override := os.Getenv("SERVER_MONITOR_CONFIG"); override != "" {
		path = override
		err = nil
	}
	a.configPath = path
	cfg := monitor.DefaultConfig()
	if err == nil {
		cfg, err = monitor.LoadConfig(path)
	}
	if err != nil {
		a.startupError = err.Error()
		cfg = monitor.DefaultConfig()
	}
	a.ssh, err = monitor.NewSSHClient()
	if err != nil {
		a.startupError = err.Error()
		return
	}
	a.engine = monitor.NewEngine(a.ssh.NewSession, func(s monitor.Snapshot) {
		setTrayTitle(monitor.MenuTitle(s))
		runtime.EventsEmit(a.ctx, "monitor:snapshot", s)
	})
	a.engine.Apply(cfg)
}
func (a *App) domReady(ctx context.Context) {
	if a.engine == nil || len(a.engine.Config().Servers) == 0 || a.startupError != "" {
		a.ShowMainWindow()
	}
}
func (a *App) shutdown(ctx context.Context) {
	if a.engine != nil {
		a.engine.Close()
	}
	if a.ssh != nil {
		a.ssh.Close()
	}
}
func (a *App) GetInitialState() InitialState {
	a.mu.Lock()
	mode := a.mode
	a.mu.Unlock()
	state := InitialState{Config: monitor.DefaultConfig(), Mode: mode, Error: a.startupError, Version: appVersion}
	if a.engine != nil {
		state.Config = a.engine.Config()
		state.Snapshot = a.engine.Snapshot()
	}
	return state
}
func (a *App) SaveConfiguration(c monitor.Config) (monitor.Config, error) {
	a.saveMu.Lock()
	defer a.saveMu.Unlock()
	if a.startupError != "" {
		return c, fmt.Errorf("启动失败，原配置已保留：%s", a.startupError)
	}
	if a.engine == nil {
		return c, fmt.Errorf("监控服务未启动")
	}
	c, err := monitor.NormalizeConfig(c)
	if err != nil {
		return c, err
	}
	if err = monitor.SaveConfig(a.configPath, c); err != nil {
		return c, err
	}
	a.engine.Apply(c)
	return c, nil
}
func (a *App) ListSSHConfigConnections() ([]monitor.SSHChoice, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return monitor.DiscoverSSHConfig(filepath.Join(home, ".ssh", "config"))
}
func (a *App) RetryServer(id string) error {
	if a.engine == nil {
		return fmt.Errorf("监控服务未启动")
	}
	return a.engine.Retry(id)
}
func (a *App) HandleWake() {
	if a.engine == nil {
		return
	}
	for _, s := range a.engine.Config().Servers {
		if s.Enabled {
			_ = a.engine.Retry(s.ID)
		}
	}
}
func (a *App) HandleTrayClick() {
	a.mu.Lock()
	visible, mode := a.visible, a.mode
	a.mu.Unlock()
	if platformAppHidden() {
		visible = false
	}
	if visible && mode == "mini" {
		a.HideWindow()
	} else {
		a.ShowMiniWindow()
	}
}
func (a *App) ShowMiniWindow() {
	runtime.WindowSetMinSize(a.ctx, 360, 300)
	runtime.WindowSetSize(a.ctx, miniWidth, miniHeight)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	x, y := popupPosition(miniWidth)
	runtime.WindowSetPosition(a.ctx, x, y)
	a.mu.Lock()
	a.mode = "mini"
	a.visible = true
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "window:mode", "mini")
	runtime.WindowShow(a.ctx)
}
func (a *App) ShowMainWindow() {
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
	runtime.WindowSetMinSize(a.ctx, 640, 480)
	runtime.WindowSetSize(a.ctx, mainWidth, mainHeight)
	runtime.WindowCenter(a.ctx)
	a.mu.Lock()
	a.mode = "main"
	a.visible = true
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "window:mode", "main")
	runtime.WindowShow(a.ctx)
}
func (a *App) HideWindow() { runtime.WindowHide(a.ctx); a.mu.Lock(); a.visible = false; a.mu.Unlock() }
func (a *App) Quit()       { runtime.Quit(a.ctx) }
