//go:build windows

package command

import "os/exec"

// setProcessGroup is a no-op on Windows.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the process started by cmd. Windows has no process
// groups to signal, so only the direct child is stopped.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
