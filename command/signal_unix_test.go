//go:build unix

package command

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMainCancelsContextOnSIGINT(t *testing.T) {
	exits := newExitRecorder()
	started := make(chan struct{})

	commands := map[string]*testCommand{
		"wait": {name: "wait", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			close(started)
			select {
			case <-c.Context.Done():
				return ExitCodeInterrupt
			case <-time.After(5 * time.Second):
				return 0
			}
		}},
	}

	go func() {
		<-started
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	got := runTestMain(t, []string{"wait"}, commands, withExit(exits.exit), WithGracePeriod(0))
	if got.code != ExitCodeInterrupt {
		t.Fatalf("exit code = %d", got.code)
	}
	if !strings.Contains(got.stderr, interruptMessage) {
		t.Fatalf("stderr = %q", got.stderr)
	}
	if exits.count() != 0 {
		t.Fatal("a command that stops on its own should not be force-exited")
	}
}
