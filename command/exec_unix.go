//go:build unix

package command

import (
	"os/exec"
	"syscall"
	"time"
)

// processKillGrace is how long a stopped process group has to exit after
// SIGTERM before it is killed.
const processKillGrace = 2 * time.Second

// setProcessGroup starts cmd in its own process group, so it and every
// process it starts can be stopped together.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGTERM to the process group of cmd, then SIGKILL
// if it is still running after a short grace period.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	pgid := -cmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGTERM)

	go func() {
		time.Sleep(processKillGrace)
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	}()
}
