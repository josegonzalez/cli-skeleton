package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock replaces timeNow with a clock that advances by step on every
// call.
func fakeClock(t *testing.T, step time.Duration) {
	t.Helper()
	var mu sync.Mutex
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	original := timeNow
	timeNow = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(step)
		return now
	}
	t.Cleanup(func() { timeNow = original })
}

// newStepsCommand returns a test command whose steps use the given mode and
// write their log to a temp state dir.
func newStepsCommand(t *testing.T, mode string, settings *Settings) (*testCommand, *bytes.Buffer) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("TEST_APP_DEBUG_LOG", "")
	c, _, stderr := newExecuteCommand(settings)
	c.progressModeOverride = mode
	c.commandName = "eat"
	c.progressState = &progressState{}
	t.Cleanup(c.progressState.close)
	return c, stderr
}

func TestStepsPlain(t *testing.T) {
	fakeClock(t, 100*time.Millisecond)
	c, stderr := newStepsCommand(t, progressPlain, nil)

	steps := c.Steps(2)
	err := steps.Run("Unwrapping", func(step *Step) error {
		step.Printf("removing wrapper")
		step.Result("1 wrapper")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := steps.Run("Eating", func(step *Step) error { return nil }); err != nil {
		t.Fatal(err)
	}

	out := stderr.String()
	for _, want := range []string{"-----> [1/2] Unwrapping\n", "       1 wrapper (", "-----> [2/2] Eating\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "removing wrapper") {
		t.Error("step output should only be shown on failure in plain mode")
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("plain mode should not draw escape codes")
	}
}

func TestStepsFailure(t *testing.T) {
	fakeClock(t, time.Second)
	c, stderr := newStepsCommand(t, progressPlain, nil)
	c.supportURL = "https://example.com/help"

	code := Execute(c, nil, func(map[string]Argument) error {
		return c.Steps(1).Run("Eating", func(step *Step) error {
			step.Printf("chewing")
			step.Write([]byte("swallowing without a newline"))
			return errors.New("lollipop stuck")
		})
	})

	if code != ExitCodeError {
		t.Fatalf("exit code = %d", code)
	}

	out := stderr.String()
	for _, want := range []string{
		" !     [1/1] Eating failed after",
		"Output from this step:\n       chewing\n       swallowing without a newline\n",
		" !     lollipop stuck\n",
		" !     Full log: ",
		" !     Help: https://example.com/help\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "lollipop stuck") != 1 {
		t.Errorf("Execute should not print the step error again:\n%s", out)
	}

	start := strings.Index(out, "Full log: ") + len("Full log: ")
	path := strings.TrimSpace(strings.SplitN(out[start:], "\n", 2)[0])
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the log: %v", err)
	}
	for _, want := range []string{"==> step start [1/1] Eating", "chewing", "swallowing without a newline", "==> step end [1/1] Eating"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("log missing %q:\n%s", want, data)
		}
	}
	if !strings.Contains(filepath.Base(path), "test-app-eat-") {
		t.Errorf("log name = %q", path)
	}
}

func TestStepsVerbose(t *testing.T) {
	fakeClock(t, 100*time.Millisecond)
	c, stderr := newStepsCommand(t, progressVerbose, nil)

	c.Steps(1).Run("Eating", func(step *Step) error {
		step.Printf("chewing")
		return nil
	})

	out := stderr.String()
	if !strings.Contains(out, "-----> [1/1] Eating\n") || !strings.Contains(out, "s chewing\n") || !strings.Contains(out, "       +0.") {
		t.Fatalf("verbose output = %q", out)
	}
}

func TestStepsQuiet(t *testing.T) {
	c, stderr := newStepsCommand(t, progressQuiet, nil)

	c.Steps(1).Run("Eating", func(step *Step) error { return nil })
	if stderr.Len() != 0 {
		t.Fatalf("quiet mode should print nothing on success: %q", stderr.String())
	}

	c.Steps(1).Run("Eating", func(step *Step) error {
		step.Printf("chewing")
		return errors.New("stuck")
	})
	if !strings.Contains(stderr.String(), "failed after") || !strings.Contains(stderr.String(), "chewing") {
		t.Fatalf("quiet mode should still show failures: %q", stderr.String())
	}
}

func TestStepsModes(t *testing.T) {
	tests := []struct {
		settings *Settings
		want     string
	}{
		{settings: &Settings{}, want: progressPlain},
		{settings: &Settings{Quiet: true}, want: progressQuiet},
		{settings: &Settings{Verbose: true}, want: progressVerbose},
		{settings: &Settings{Debug: true}, want: progressVerbose},
	}
	for _, tt := range tests {
		c, _, _ := newExecuteCommand(tt.settings)
		if got := c.progressMode(); got != tt.want {
			t.Errorf("progressMode(%+v) = %q, want %q", *tt.settings, got, tt.want)
		}
	}
}

func TestStepsFancy(t *testing.T) {
	fakeClock(t, 100*time.Millisecond)
	t.Setenv("COLUMNS", "60")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	c, stderr := newStepsCommand(t, progressFancy, nil)

	err := c.Steps(1).Run("Eating", func(step *Step) error {
		for i := 0; i < 6; i++ {
			step.Printf("line %d \x1b[31mred\x1b[0m\tend", i)
		}
		step.Printf("%s", strings.Repeat("x", 200))
		step.r.mu.Lock()
		step.r.drawLocked()
		step.r.mu.Unlock()
		step.Result("ate it")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	out := stderr.String()
	for _, want := range []string{"\x1b[?25l", "----", "[1/1] Eating", "line 5 red    end", "\x1b[J", "\x1b[?25h", "-----> [1/1] Eating (", "       ate it\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%q", want, out)
		}
	}
	if strings.Contains(out, "line 1 ") {
		t.Error("only the last lines should be shown in the tail")
	}
	if strings.Contains(out, strings.Repeat("x", 60)) {
		t.Error("tail lines should be truncated to the terminal width")
	}
	if strings.Contains(out, "\x1b[31m") {
		t.Error("escape codes in step output should be removed")
	}
}

func TestStepsZeroTailLines(t *testing.T) {
	c, stderr := newStepsCommand(t, progressFancy, nil)
	zero := 0
	c.tailLines = &zero

	c.Steps(1).Run("Eating", func(step *Step) error {
		step.Printf("hidden")
		step.r.mu.Lock()
		step.r.drawLocked()
		step.r.mu.Unlock()
		return nil
	})
	if strings.Contains(stderr.String(), "hidden") {
		t.Fatalf("no tail lines should be shown: %q", stderr.String())
	}
}

func TestStepsSummary(t *testing.T) {
	fakeClock(t, time.Second)
	c, stderr := newStepsCommand(t, progressPlain, nil)

	steps := c.Steps(4)
	for _, title := range []string{"a", "b", "c", "d"} {
		steps.Run(title, func(*Step) error { return nil })
	}
	steps.Summary()

	out := stderr.String()
	if !strings.Contains(out, "Finished in ") || !strings.Contains(out, "Slowest steps: ") {
		t.Fatalf("summary = %q", out)
	}
	line := out[strings.Index(out, "Slowest steps: "):]
	if strings.Count(line, ",") != 2 {
		t.Fatalf("summary should list three steps: %q", line)
	}
}

func TestSpinner(t *testing.T) {
	c, stderr := newStepsCommand(t, progressPlain, nil)

	spinner := c.Spinner("Fetching")
	spinner.Printf("connecting")
	if err := spinner.Done(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "-----> Fetching\n") || strings.Contains(stderr.String(), "[") {
		t.Fatalf("spinner output = %q", stderr.String())
	}
}

func TestProgressBarPlain(t *testing.T) {
	c, stderr := newStepsCommand(t, progressPlain, nil)

	bar := c.ProgressBar("Downloading", 50)
	for i := 0; i < 50; i++ {
		bar.Add(1)
	}
	bar.Add(10)
	bar.Done(nil)

	out := stderr.String()
	for _, want := range []string{"Downloading 0% (0/50)", "Downloading 50% (25/50)", "Downloading 100% (50/50)"} {
		if !strings.Contains(out, want) && want != "Downloading 0% (0/50)" {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "Downloading 50%") != 1 {
		t.Errorf("each 10%% should be printed once:\n%s", out)
	}
}

func TestProgressBarRender(t *testing.T) {
	bar := &barState{current: 25, total: 100}
	got := bar.render(10*time.Second, false)
	if !strings.Contains(got, "=====---------------") || !strings.Contains(got, " 25% 25/100") || !strings.Contains(got, "ETA 30s") {
		t.Fatalf("render = %q", got)
	}

	done := &barState{current: 100, total: 100}
	if got := done.render(time.Second, true); strings.Contains(got, "ETA") || !strings.Contains(got, "100%") {
		t.Fatalf("finished bar = %q", got)
	}
}

func TestConcurrentSteps(t *testing.T) {
	c, stderr := newStepsCommand(t, progressVerbose, nil)

	var wg sync.WaitGroup
	for _, title := range []string{"red", "blue", "green"} {
		wg.Add(1)
		go func(title string) {
			defer wg.Done()
			s := c.Spinner(title)
			for i := 0; i < 20; i++ {
				s.Printf("%s line %d", title, i)
			}
			s.Done(nil)
		}(title)
	}
	wg.Wait()

	for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		if strings.Count(line, " line ") > 1 {
			t.Fatalf("output lines should not interleave: %q", line)
		}
	}
}

func TestStepsInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, stderr := newStepsCommand(t, progressFancy, nil)
	c.Context = ctx
	c.interrupts = newInterruptHandler(cancel, func(string) {}, 0, func(int) {})
	defer c.interrupts.stop()

	err := c.Steps(1).Run("Eating", func(step *Step) error {
		step.Printf("chewing")
		c.interrupts.interrupt()
		<-ctx.Done()
		return ctx.Err()
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	out := stderr.String()
	if !strings.Contains(out, "\x1b[?25h") || !strings.Contains(out, "Full log: ") {
		t.Fatalf("interrupt should restore the cursor and point to the log: %q", out)
	}
	if strings.Contains(out, "failed after") {
		t.Fatal("an interrupted step is not a failure")
	}
}

func TestStepsDebugLog(t *testing.T) {
	c, _ := newStepsCommand(t, progressQuiet, nil)
	var log bytes.Buffer
	c.debugLog = &log

	c.Steps(1).Run("Eating", func(step *Step) error {
		step.Printf("chewing")
		return nil
	})
	if !strings.Contains(log.String(), "chewing") {
		t.Fatalf("step output should go to <APP>_DEBUG_LOG: %q", log.String())
	}
}

func TestCountdown(t *testing.T) {
	c, stderr := newStepsCommand(t, progressPlain, nil)
	c.Meta.flags.force = true
	if err := c.Countdown("Deleting in a moment", 10); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("--force should skip the countdown: %q", stderr.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, stderr = newStepsCommand(t, progressPlain, nil)
	c.Context = ctx
	if err := c.Countdown("Deleting in a moment", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(stderr.String(), "Deleting in a moment") || !strings.Contains(stderr.String(), "Continuing in 10s") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSanitize(t *testing.T) {
	tests := map[string]string{
		"plain":                       "plain",
		"\x1b[1;31mred\x1b[0m":        "red",
		"10%\r50%\r100%":              "100%",
		"trailing\r":                  "trailing",
		"tab\there":                   "tab    here",
		"bell\a and \x1b[?25lcursor":  "bell and cursor",
		"progress 10%\r":              "progress 10%",
		"unicode ✓ stays":             "unicode ✓ stays",
		"\x1b[2K\rcleared then shown": "cleared then shown",
	}
	for in, want := range tests {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := map[time.Duration]string{
		400 * time.Millisecond: "0.4s",
		12 * time.Second:       "12s",
		185 * time.Second:      "3m05s",
	}
	for in, want := range tests {
		if got := formatDuration(in); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 4, true); got != "abc…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abcdef", 4, false); got != "abc." {
		t.Errorf("ascii truncate = %q", got)
	}
	if got := truncate("abc", 4, true); got != "abc" {
		t.Errorf("short truncate = %q", got)
	}
}
