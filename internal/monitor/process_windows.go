package monitor

import (
	"os/exec"
	"syscall"
	"time"
)

// Windows UI/packaging remains a later milestone; collector code is kept portable.
func prepareCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.WaitDelay = time.Second
}
