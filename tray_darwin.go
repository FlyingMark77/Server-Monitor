package main

/*
#cgo LDFLAGS: -framework Cocoa -framework Foundation
#include <stdlib.h>
extern void smSetupTray(void);
extern void smSetTitle(const char *title);
extern void smPopupPosition(int width, int *x, int *y);
*/
import "C"
import "unsafe"

var gApp *App

//export goStatusItemClicked
func goStatusItemClicked() {
	if gApp != nil {
		go gApp.HandleTrayClick()
	}
}

//export goSystemWoke
func goSystemWoke() {
	if gApp != nil {
		go gApp.HandleWake()
	}
}
func trayRun(a *App) { gApp = a; C.smSetupTray() }
func setTrayTitle(title string) {
	s := C.CString(title)
	defer C.free(unsafe.Pointer(s))
	C.smSetTitle(s)
}
func popupPosition(width int) (int, int) {
	var x, y C.int
	C.smPopupPosition(C.int(width), &x, &y)
	return int(x), int(y)
}
