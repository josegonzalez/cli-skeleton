package command

import (
	"os/exec"
	"strings"
)

// Exec runs cmd as part of the step. Its stdout and stderr, unless already
// set, are captured as step output, and its stdin defaults to the null
// device so it cannot hang waiting for input. When the run is interrupted,
// cmd and every process it started are stopped.
func (s *Step) Exec(cmd *exec.Cmd) error {
	if cmd.Stdout == nil {
		cmd.Stdout = s
	}
	if cmd.Stderr == nil {
		cmd.Stderr = s
	}
	s.Printf("$ %s", strings.Join(cmd.Args, " "))

	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}

	ctx := s.r.m.context()
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			killProcessGroup(cmd)
		case <-done:
		}
	}()

	err := cmd.Wait()
	close(done)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
