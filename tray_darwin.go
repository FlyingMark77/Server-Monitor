package main

/*
#cgo LDFLAGS: -framework Cocoa -framework Foundation
#include <stdlib.h>
extern void smSetupTray(void);
extern void smSetTitle(const char *title);
extern void smPopupPosition(int width, int *x, int *y);
*/
import "C"
import (
	"sync/atomic"
	"unsafe"
)

var gApp atomic.Pointer[App]

//export goStatusItemClicked
func goStatusItemClicked() {
	if a := gApp.Load(); a != nil {
		go a.HandleTrayClick()
	}
}

//export goSystemWoke
func goSystemWoke() {
	if a := gApp.Load(); a != nil {
		go a.HandleWake()
	}
}
func trayRun(a *App) { gApp.Store(a); C.smSetupTray() }
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
