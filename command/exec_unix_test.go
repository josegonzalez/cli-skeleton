//go:build unix

package command

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestStepExec(t *testing.T) {
	c, stderr := newStepsCommand(t, progressVerbose, nil)

	err := c.Steps(1).Run("Running", func(step *Step) error {
		return step.Exec(exec.Command("sh", "-c", "echo out; echo err >&2; cat"))
	})
	if err != nil {
		t.Fatal(err)
	}

	out := stderr.String()
	for _, want := range []string{"$ sh -c echo out; echo err >&2; cat", " out\n", " err\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestStepExecFailure(t *testing.T) {
	c, _ := newStepsCommand(t, progressQuiet, nil)

	err := c.Steps(1).Run("Running", func(step *Step) error {
		return step.Exec(exec.Command("sh", "-c", "exit 3"))
	})
	var stepErr *StepError
	if !errors.As(err, &stepErr) {
		t.Fatalf("err = %v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("the exit error should be kept: %v", err)
	}
}

func TestStepExecStopsProcessGroupOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c, _ := newStepsCommand(t, progressQuiet, nil)
	c.Context = ctx

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := c.Steps(1).Run("Sleeping", func(step *Step) error {
		return step.Exec(exec.Command("sh", "-c", "sleep 30 & sleep 30; wait"))
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("the process group was not stopped")
	}
}
