//go:build !darwin

package main

// Platform boundary only. Windows tray and packaging are not yet implemented.
func trayRun(a *App)                     {}
func setTrayTitle(title string)          {}
func platformAppHidden() bool            { return false }
func popupPosition(width int) (int, int) { return 20, 32 }
