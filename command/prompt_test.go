package command

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeTerminal makes stdin readers count as terminals for the test.
func fakeTerminal(t *testing.T) {
	t.Helper()
	original := isTerminalReader
	isTerminalReader = func(io.Reader) bool { return true }
	t.Cleanup(func() { isTerminalReader = original })
}

// newPromptCommand returns a test command whose stdin holds input.
func newPromptCommand(input string, settings *Settings) (*testCommand, *strings.Reader) {
	c, _, _ := newExecuteCommand(settings)
	stdin := strings.NewReader(input)
	inner := c.Ui.(*settingsUi)
	inner.Ui.(*TerminalUi).Reader = stdin
	inner.stdin = stdin
	c.stdin = stdin
	return c, stdin
}

func TestConfirm(t *testing.T) {
	fakeTerminal(t)

	tests := []struct {
		input string
		want  bool
	}{
		{input: "y\n", want: true},
		{input: "YES\n", want: true},
		{input: "n\n", want: false},
		{input: "\n", want: false},
		{input: "maybe\n", want: false},
	}
	for _, tt := range tests {
		c, _ := newPromptCommand(tt.input, nil)
		got, err := c.Confirm("Eat the lollipop?")
		if err != nil {
			t.Fatalf("%q: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("Confirm with %q = %v, want %v", tt.input, got, tt.want)
		}
	}

	c, _ := newPromptCommand("", nil)
	stderr := c.stderr.(interface{ String() string })
	c.Confirm("Eat the lollipop?")
	if !strings.Contains(stderr.String(), "Eat the lollipop? [y/N]") {
		t.Errorf("prompt = %q", stderr.String())
	}
}

func TestConfirmForce(t *testing.T) {
	c, _ := newPromptCommand("", nil)
	c.Meta.flags.force = true

	got, err := c.Confirm("Eat the lollipop?")
	if err != nil || !got {
		t.Fatalf("--force should confirm without asking: %v %v", got, err)
	}
}

func TestConfirmWithoutTerminal(t *testing.T) {
	c, _ := newPromptCommand("y\n", nil)
	c.FlagSet()
	c.Meta.FlagSet("eat", FlagSetForce)

	got, err := c.Confirm("Eat the lollipop?")
	var userErr *UserError
	if got || !errors.As(err, &userErr) {
		t.Fatalf("expected a UserError, got %v %v", got, err)
	}
	if !strings.Contains(userErr.Message, "stdin is not a terminal") || !strings.Contains(userErr.Hint, "--force") {
		t.Fatalf("error = %+v", userErr)
	}
	if userErr.exitCode() != ExitCodeUsage {
		t.Fatalf("exit code = %d", userErr.exitCode())
	}
}

func TestConfirmWithNoInput(t *testing.T) {
	fakeTerminal(t)
	c, _ := newPromptCommand("y\n", &Settings{NoInput: true})

	_, err := c.Confirm("Eat the lollipop?")
	var userErr *UserError
	if !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "--no-input") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(userErr.Hint, "--force") {
		t.Fatal("the hint should only mention --force when the command accepts it")
	}
}

func TestConfirmName(t *testing.T) {
	c, _ := newPromptCommand("", nil)
	c.Meta.flags.confirm = "my-app"
	if err := c.ConfirmName("my-app"); err != nil {
		t.Fatalf("--confirm should match: %v", err)
	}

	c.Meta.flags.confirm = "other-app"
	var usageErr *UsageError
	if err := c.ConfirmName("my-app"); !errors.As(err, &usageErr) {
		t.Fatalf("a mismatched --confirm should be a usage error: %v", err)
	}

	fakeTerminal(t)
	c, _ = newPromptCommand("my-app\n", nil)
	if err := c.ConfirmName("my-app"); err != nil {
		t.Fatalf("typed confirmation should match: %v", err)
	}
	stderr := c.stderr.(interface{ String() string })
	if !strings.Contains(stderr.String(), `Type "my-app" to confirm`) || !strings.Contains(stderr.String(), cancelHint) {
		t.Fatalf("prompt = %q", stderr.String())
	}

	c, _ = newPromptCommand("my-ap\n", nil)
	var userErr *UserError
	if err := c.ConfirmName("my-app"); !errors.As(err, &userErr) || !strings.Contains(userErr.Message, "nothing was changed") {
		t.Fatalf("typed mismatch should fail: %v", err)
	}
}

func TestConfirmNameWithoutTerminal(t *testing.T) {
	c, _ := newPromptCommand("my-app\n", nil)
	c.Meta.FlagSet("eat", FlagSetConfirm)

	var userErr *UserError
	if err := c.ConfirmName("my-app"); !errors.As(err, &userErr) || !strings.Contains(userErr.Hint, "--confirm=my-app") {
		t.Fatalf("error = %v", err)
	}
}

func TestArgumentPrompt(t *testing.T) {
	fakeTerminal(t)
	c, _ := newPromptCommand("grape\n", nil)
	c.args = []Argument{{Name: "flavor", Type: ArgumentString, Prompt: "Which flavor?"}}

	var flavor string
	code := Execute(c, nil, func(arguments map[string]Argument) error {
		flavor = arguments["flavor"].StringValue()
		return nil
	})
	if code != ExitCodeOK || flavor != "grape" {
		t.Fatalf("code = %d, flavor = %q", code, flavor)
	}

	stderr := c.stderr.(interface{ String() string })
	if !strings.Contains(stderr.String(), "Which flavor? "+cancelHint) {
		t.Fatalf("prompt = %q", stderr.String())
	}
}

func TestArgumentPromptEmptyAnswer(t *testing.T) {
	fakeTerminal(t)
	c, _ := newPromptCommand("\n", nil)
	c.args = []Argument{{Name: "flavor", Type: ArgumentString, Prompt: "Which flavor?"}}

	code := Execute(c, nil, func(map[string]Argument) error { return nil })
	if code != ExitCodeUsage {
		t.Fatalf("code = %d", code)
	}
}

func TestArgumentPromptWithoutTerminal(t *testing.T) {
	c, _ := newPromptCommand("grape\n", nil)
	c.args = []Argument{{Name: "flavor", Type: ArgumentString, Prompt: "Which flavor?"}}

	code := Execute(c, nil, func(map[string]Argument) error {
		t.Fatal("run should not be called")
		return nil
	})
	if code != ExitCodeUsage {
		t.Fatalf("code = %d", code)
	}

	stderr := c.stderr.(interface{ String() string })
	if strings.Contains(stderr.String(), "Which flavor?") {
		t.Fatal("should not prompt without a terminal")
	}
	if !strings.Contains(stderr.String(), "<flavor>") || !strings.Contains(stderr.String(), "USAGE") {
		t.Fatalf("expected concise help naming the argument: %q", stderr.String())
	}
}

func TestArgumentValidate(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	c.args = []Argument{{
		Name: "flavor",
		Type: ArgumentString,
		Validate: func(value string) error {
			if value != "grape" && value != "cherry" {
				return errors.New("must be grape or cherry")
			}
			return nil
		},
	}}

	code := Execute(c, []string{"lemon"}, func(map[string]Argument) error { return nil })
	if code != ExitCodeUsage || !strings.Contains(stderr.String(), "Invalid value for argument flavor: must be grape or cherry") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}

	c, _, _ = newExecuteCommand(nil)
	c.args = []Argument{{Name: "flavors", Type: ArgumentList, Validate: func(value string) error {
		if value == "lemon" {
			return errors.New("no lemons")
		}
		return nil
	}}}
	if code := Execute(c, []string{"grape", "lemon"}, func(map[string]Argument) error { return nil }); code != ExitCodeUsage {
		t.Fatalf("list values should each be validated: %d", code)
	}
}

func TestNewArgumentTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ParseArguments([]string{path, "1m30s", "2.5"}, []Argument{
		{Name: "file", Type: ArgumentFile},
		{Name: "wait", Type: ArgumentDuration},
		{Name: "ratio", Type: ArgumentFloat},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got["file"].PathValue() != path {
		t.Errorf("PathValue = %q", got["file"].PathValue())
	}
	r, err := got["file"].FileValue()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(r)
	r.Close()
	if string(data) != "contents" {
		t.Errorf("FileValue read %q", data)
	}
	if got["wait"].DurationValue() != 90*time.Second {
		t.Errorf("DurationValue = %v", got["wait"].DurationValue())
	}
	if got["ratio"].FloatValue() != 2.5 {
		t.Errorf("FloatValue = %v", got["ratio"].FloatValue())
	}

	if _, err := ParseArguments([]string{"-"}, []Argument{{Name: "file", Type: ArgumentFile}}); err != nil {
		t.Errorf("- should be accepted as stdin: %v", err)
	}

	failures := map[string][]Argument{
		filepath.Join(t.TempDir(), "missing"): {{Name: "file", Type: ArgumentFile}},
		t.TempDir():                           {{Name: "file", Type: ArgumentFile}},
		"soon":                                {{Name: "wait", Type: ArgumentDuration}},
		"half":                                {{Name: "ratio", Type: ArgumentFloat}},
	}
	for value, arguments := range failures {
		if _, err := ParseArguments([]string{value}, arguments); err == nil {
			t.Errorf("expected an error for %q as %s", value, arguments[0].Name)
		}
	}

	defaults, err := ParseArguments(nil, []Argument{
		{Name: "file", Type: ArgumentFile, Optional: true},
		{Name: "wait", Type: ArgumentDuration, Optional: true},
		{Name: "ratio", Type: ArgumentFloat, Optional: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if defaults["file"].PathValue() != "" || defaults["wait"].DurationValue() != 0 || defaults["ratio"].FloatValue() != 0 {
		t.Errorf("zero values not set: %+v", defaults)
	}

	help := ArgumentsString([]Argument{
		{Name: "file", Type: ArgumentFile},
		{Name: "wait", Type: ArgumentDuration},
		{Name: "ratio", Type: ArgumentFloat},
	})
	for _, want := range []string{"<file> file", "<wait> duration", "<ratio> float"} {
		if !strings.Contains(help, want) {
			t.Errorf("help missing %q: %q", want, help)
		}
	}
}

func TestRequireStdin(t *testing.T) {
	c, _ := newPromptCommand("", nil)
	if err := c.RequireStdin(); err != nil {
		t.Fatalf("piped stdin should be accepted: %v", err)
	}

	fakeTerminal(t)
	c, _ = newPromptCommand("", nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		return c.RequireStdin()
	})
	stderr := c.stderr.(interface{ String() string })
	if code != ExitCodeUsage || !strings.Contains(stderr.String(), "nothing was piped") || !strings.Contains(stderr.String(), "USAGE") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestInterruptDuringPromptExitsImmediately(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	exits := newExitRecorder()
	warns := &warnRecorder{}
	h := newInterruptHandler(cancel, warns.warn, 0, exits.exit)
	defer h.stop()

	inner, _, stderr := newTestTerminalUi("", false)
	ui := newSettingsUi(inner, &Settings{}, nil)
	ui.interrupts = h
	ui.stderr = stderr

	done := ui.whilePrompting()
	h.interrupt()
	done()

	if code := <-exits.ch; code != ExitCodeInterrupt {
		t.Fatalf("exit code = %d", code)
	}
	if len(warns.all()) != 0 {
		t.Fatal("an interrupt during a prompt should not print the graceful-stop message")
	}
	if stderr.String() != "\n" {
		t.Fatalf("expected a newline to end the prompt line, got %q", stderr.String())
	}
}

func TestMainAsksToRunSuggestion(t *testing.T) {
	fakeTerminal(t)

	var ran bool
	commands := map[string]*testCommand{
		"eat": {name: "eat", run: func(*testCommand, map[string]Argument) int {
			ran = true
			return 0
		}},
	}

	got := runTestMainWithInput(t, "y\n", []string{"eatt"}, commands)
	if got.code != 0 || !ran {
		t.Fatalf("code = %d, ran = %v, stderr = %q", got.code, ran, got.stderr)
	}
	if !strings.Contains(got.stderr, `Did you mean "eat"? [y/N]`) {
		t.Fatalf("stderr = %q", got.stderr)
	}

	ran = false
	got = runTestMainWithInput(t, "n\n", []string{"eatt"}, commands)
	if got.code != ExitCodeUnknownCommand || ran {
		t.Fatalf("declining should not run the command: code = %d, ran = %v", got.code, ran)
	}

	got = runTestMainWithInput(t, "y\n", []string{"--no-input", "eatt"}, commands)
	if got.code != ExitCodeUnknownCommand || ran || strings.Contains(got.stderr, "[y/N]") {
		t.Fatalf("--no-input should not prompt: code = %d, stderr = %q", got.code, got.stderr)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nano")
	if got := EditorCommand(); got != "nano" {
		t.Errorf("EditorCommand = %q", got)
	}

	t.Setenv("VISUAL", "code --wait")
	if got := EditorCommand(); got != "code --wait" {
		t.Errorf("VISUAL should win: %q", got)
	}
}

func TestShell(t *testing.T) {
	m := &Meta{}
	t.Setenv("SHELL", "/bin/zsh")
	if got := m.Shell(); got != "/bin/zsh" {
		t.Errorf("Shell = %q", got)
	}
}

func TestEditorAndShellRequireTerminal(t *testing.T) {
	c, _ := newPromptCommand("", nil)

	var userErr *UserError
	if _, err := c.Editor("text"); !errors.As(err, &userErr) {
		t.Errorf("Editor without a terminal: %v", err)
	}
	if err := c.RunShell(context.Background(), nil); !errors.As(err, &userErr) {
		t.Errorf("RunShell without a terminal: %v", err)
	}
}
