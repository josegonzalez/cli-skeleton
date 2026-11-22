package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

// newExecuteCommand returns a test command whose Meta writes to buffers.
func newExecuteCommand(settings *Settings) (*testCommand, *bytes.Buffer, *bytes.Buffer) {
	ui, stdout, stderr := newTestTerminalUi("", false)
	if settings == nil {
		settings = &Settings{}
	}
	c := &testCommand{
		name:  "eat",
		flags: FlagSetClient,
		args:  []Argument{{Name: "speed", Type: ArgumentString, Optional: true}},
		Meta: Meta{
			Ui:       newSettingsUi(ui, settings, nil),
			appName:  "test-app",
			version:  "1.2.3",
			settings: settings,
			stderr:   stderr,
			stdout:   stdout,
		},
	}
	return c, stdout, stderr
}

func TestExecuteSuccess(t *testing.T) {
	c, stdout, stderr := newExecuteCommand(nil)
	var count int
	c.extra = func(f *flag.FlagSet) { f.IntVar(&count, "count", 1, "") }

	code := Execute(c, []string{"--count", "3", "slowly"}, func(arguments map[string]Argument) error {
		c.Ui.Output(fmt.Sprintf("%d %s", count, arguments["speed"].StringValue()))
		return nil
	})

	if code != ExitCodeOK {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "3 slowly\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecuteHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}} {
		c, stdout, stderr := newExecuteCommand(nil)
		code := Execute(c, args, func(map[string]Argument) error {
			t.Fatal("run should not be called for help")
			return nil
		})

		if code != ExitCodeOK {
			t.Errorf("%v: exit code = %d", args, code)
		}
		if !strings.Contains(stdout.String(), "test-app eat") || stderr.Len() != 0 {
			t.Errorf("%v: stdout = %q, stderr = %q", args, stdout.String(), stderr.String())
		}
	}
}

func TestExecuteUsageErrors(t *testing.T) {
	tests := map[string][]string{
		"unknown flag":   {"--unknown"},
		"too many args":  {"a", "b"},
		"missing value":  {"--count"},
		"invalid number": {"--count", "x"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			c, stdout, stderr := newExecuteCommand(nil)
			c.extra = func(f *flag.FlagSet) { f.Int("count", 1, "") }

			code := Execute(c, args, func(map[string]Argument) error {
				t.Fatal("run should not be called")
				return nil
			})

			if code != ExitCodeUsage {
				t.Fatalf("exit code = %d", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q", stdout.String())
			}
			lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("expected an error and a hint, got %q", stderr.String())
			}
			if lines[1] != "For additional help try 'test-app eat --help'" {
				t.Fatalf("hint = %q", lines[1])
			}
		})
	}
}

func TestExecuteUsageErrorCustomHint(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return &UsageError{Err: errors.New("--color must be red or blue"), Hint: "Pass --color red"}
	})

	if code != ExitCodeUsage {
		t.Fatalf("exit code = %d", code)
	}
	if stderr.String() != "--color must be red or blue\nPass --color red\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteUserError(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return fmt.Errorf("wrapped: %w", &UserError{
			Message: "Can't write to file.txt",
			Hint:    "Try running 'chmod +w file.txt'",
			Code:    4,
			Err:     errors.New("permission denied"),
		})
	})

	if code != 4 {
		t.Fatalf("exit code = %d", code)
	}
	want := "Can't write to file.txt\nTry running 'chmod +w file.txt'\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestExecuteUserErrorDebugDetails(t *testing.T) {
	c, _, stderr := newExecuteCommand(&Settings{Debug: true})
	Execute(c, nil, func(map[string]Argument) error {
		return &UserError{Message: "Can't write to file.txt", Err: errors.New("permission denied")}
	})

	if !strings.Contains(stderr.String(), "permission denied") {
		t.Fatalf("debug details missing: %q", stderr.String())
	}
}

func TestExecuteMultiError(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		errs := NewMultiError("Unable to eat 2 lollipops:")
		errs.Append(errors.New("red: out of stock"))
		errs.Append(nil)
		errs.Append(errors.New("blue: out of stock"))
		return errs.ErrorOrNil()
	})

	if code != ExitCodeError {
		t.Fatalf("exit code = %d", code)
	}
	want := "Unable to eat 2 lollipops:\n  red: out of stock\n  blue: out of stock\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}

	if NewMultiError("empty").ErrorOrNil() != nil {
		t.Fatal("an empty MultiError should be nil")
	}
}

func TestExecuteCanceled(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return fmt.Errorf("eating: %w", context.Canceled)
	})

	if code != ExitCodeInterrupt {
		t.Fatalf("exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecutePlainError(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return errors.New("out of lollipops")
	})

	if code != ExitCodeError || stderr.String() != "out of lollipops\n" {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestExecutePanic(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	c.issueURL = "https://github.com/org/repo/issues/new"

	code := Execute(c, nil, func(map[string]Argument) error {
		panic("boom")
	})

	if code != ExitCodeError {
		t.Fatalf("exit code = %d", code)
	}

	out := stderr.String()
	if !strings.Contains(out, "An unexpected error occurred while running 'test-app eat'.") {
		t.Fatalf("stderr = %q", out)
	}
	if strings.Contains(out, "goroutine") {
		t.Fatal("stack traces should only be shown under --debug")
	}
	if !strings.Contains(out, "--debug") {
		t.Fatal("expected a pointer to --debug")
	}

	start := strings.Index(out, "https://")
	if start < 0 {
		t.Fatalf("issue URL missing: %q", out)
	}
	link, err := url.Parse(strings.Fields(out[start:])[0])
	if err != nil {
		t.Fatal(err)
	}
	body := link.Query().Get("body")
	for _, want := range []string{"test-app eat", "1.2.3", "boom"} {
		if !strings.Contains(body, want) {
			t.Errorf("issue body missing %q: %q", want, body)
		}
	}
	if !strings.Contains(link.Query().Get("title"), "test-app eat") {
		t.Errorf("issue title = %q", link.Query().Get("title"))
	}
}

func TestExecutePanicDebug(t *testing.T) {
	c, _, stderr := newExecuteCommand(&Settings{Debug: true})
	Execute(c, nil, func(map[string]Argument) error {
		panic(errors.New("boom"))
	})

	out := stderr.String()
	if !strings.Contains(out, "boom") || !strings.Contains(out, "goroutine") {
		t.Fatalf("debug output should include the error and stack: %q", out)
	}
	if strings.Contains(out, "Run the command again with --debug") {
		t.Fatal("the --debug pointer should be omitted under --debug")
	}
}

func TestExecuteInternalError(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return Internal(errors.New("invariant violated"))
	})

	if code != ExitCodeError || !strings.Contains(stderr.String(), "unexpected error") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "invariant violated") {
		t.Fatal("internal error details should only be shown under --debug")
	}
	if Internal(nil) != nil {
		t.Fatal("Internal(nil) should be nil")
	}
}

func TestDebugf(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	c.Debugf("hidden %d", 1)
	if stderr.Len() != 0 {
		t.Fatalf("debug output without --debug: %q", stderr.String())
	}

	c.settings.Debug = true
	c.Debugf("shown %d", 2)
	if stderr.String() != "shown 2\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDebugLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	t.Setenv("TEST_APP_DEBUG_LOG", path)

	f, err := openDebugLog("test-app")
	if err != nil {
		t.Fatal(err)
	}

	c, _, stderr := newExecuteCommand(nil)
	c.debugLog = f
	c.Debugf("first\nsecond")
	f.Close()

	if stderr.Len() != 0 {
		t.Fatalf("debug output should go to the log file: %q", stderr.String())
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], " first") || !strings.HasSuffix(lines[1], " second") {
		t.Fatalf("log = %q", contents)
	}
}

func TestMainRecoversPanics(t *testing.T) {
	commands := map[string]*testCommand{
		"eat": {name: "eat", run: func(c *testCommand, _ map[string]Argument) int {
			panic("boom")
		}},
	}

	got := runTestMain(t, []string{"eat"}, commands, WithIssueURL("https://example.com/issues/new"))
	if got.code != ExitCodeError {
		t.Fatalf("exit code = %d", got.code)
	}
	if !strings.Contains(got.stderr, "An unexpected error occurred") || !strings.Contains(got.stderr, "https://example.com/issues/new?") {
		t.Fatalf("stderr = %q", got.stderr)
	}
}

func TestBuildInfo(t *testing.T) {
	m := &Meta{version: "1.2.3"}
	info := m.BuildInfo()
	if info.Version != "1.2.3" || info.GoVersion == "" || info.OS == "" || info.Arch == "" {
		t.Fatalf("BuildInfo = %+v", info)
	}
}

func TestExecuteExitStatus(t *testing.T) {
	c, stdout, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return &ExitStatus{Code: 42}
	})
	if code != 42 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}
