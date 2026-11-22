package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
)

func TestStandardFlags(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	c.flags = FlagSetDefault
	c.defaultPort = 8080

	code := Execute(c, []string{"--json", "-f", "-n", "--confirm", "app", "--timeout", "5s", "-a", "-u", "jose"}, func(map[string]Argument) error {
		return nil
	})
	if code != ExitCodeOK {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	if !c.IsJSON() || c.IsPlain() || !c.IsForce() || !c.IsDryRun() || !c.IsAll() {
		t.Errorf("bool flags not parsed: %+v", c.Meta.flags)
	}
	if c.Meta.flags.confirm != "app" || c.Timeout() != 5*time.Second || c.Port() != 8080 || c.User() != "jose" {
		t.Errorf("value flags not parsed: %+v", c.Meta.flags)
	}
}

func TestStandardFlagsAreOptIn(t *testing.T) {
	m := &Meta{}
	f := m.FlagSet("eat", FlagSetClient)
	for _, name := range []string{"json", "plain", "force", "dry-run", "confirm", "timeout", "output", "all", "port", "user"} {
		if f.Lookup(name) != nil {
			t.Errorf("--%s should only be added when requested", name)
		}
	}

	f = m.FlagSet("eat", FlagSetJSON)
	if f.Lookup("json") == nil || f.Lookup("quiet") != nil {
		t.Error("FlagSetJSON should only add --json")
	}

	completions := m.AutocompleteFlags(FlagSetForce)
	if _, ok := completions["-f"]; !ok {
		t.Error("completions should include -f")
	}
	if _, ok := completions["--quiet"]; ok {
		t.Error("completions should not include framework flags without FlagSetClient")
	}
}

func TestDefaultTimeout(t *testing.T) {
	m := &Meta{}
	m.FlagSet("eat", FlagSetTimeout)
	if m.Timeout() != DefaultTimeout {
		t.Fatalf("Timeout = %v", m.Timeout())
	}

	d := time.Minute
	m = &Meta{defaultTimeout: &d}
	f := m.FlagSet("eat", FlagSetTimeout)
	if m.Timeout() != time.Minute || f.Lookup("timeout").DefValue != "1m0s" {
		t.Fatalf("Timeout = %v", m.Timeout())
	}
}

func TestJSONAndPlainConflict(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	c.flags = FlagSetJSON | FlagSetPlain

	code := Execute(c, []string{"--json", "--plain"}, func(map[string]Argument) error { return nil })
	if code != ExitCodeUsage || !strings.Contains(stderr.String(), "cannot be used together") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestTimeoutContext(t *testing.T) {
	m := &Meta{Context: context.Background()}
	m.flags.timeout = 10 * time.Millisecond

	ctx, cancel := m.TimeoutContext()
	defer cancel()
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("timeout context was not cancelled")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("err = %v", ctx.Err())
	}

	m.flags.timeout = 0
	ctx, cancel = m.TimeoutContext()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("a zero timeout should not set a deadline")
	}
	cancel()
}

func TestOutputWriter(t *testing.T) {
	c, stdout, _ := newExecuteCommand(nil)

	w, err := c.OutputWriter()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "to stdout")
	w.Close()
	if stdout.String() != "to stdout" {
		t.Fatalf("stdout = %q", stdout.String())
	}

	path := filepath.Join(t.TempDir(), "out.txt")
	c.Meta.flags.output = path
	w, err = c.OutputWriter()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "to file")
	w.Close()

	data, err := os.ReadFile(path)
	if err != nil || string(data) != "to file" {
		t.Fatalf("file = %q, err = %v", data, err)
	}
}

func TestOpenInputAndOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	w, err := OpenOutput(path)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "contents")
	w.Close()

	r, err := OpenInput(path)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(r)
	r.Close()
	if string(data) != "contents" {
		t.Fatalf("read %q", data)
	}

	r, err = OpenInput("-")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stdin.Stat(); err != nil {
		t.Fatal("closing - should not close stdin")
	}

	w, err = OpenOutput("-")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stdout.Stat(); err != nil {
		t.Fatal("closing - should not close stdout")
	}
}

func TestReadSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	secret, err := ReadSecret(path)
	if err != nil || secret != "hunter2" {
		t.Fatalf("secret = %q, err = %v", secret, err)
	}

	if _, err := ReadSecret(""); err == nil {
		t.Fatal("expected an error without a path")
	}
	if _, err := ReadSecret(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestOptionalString(t *testing.T) {
	newFlags := func() (*flag.FlagSet, *OptionalStringValue) {
		f := flag.NewFlagSet("test", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		return f, OptionalString(f, "config", "default.yaml", "config file")
	}

	f, v := newFlags()
	if err := f.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if v.Value() != "default.yaml" || v.IsSet() || v.IsNone() {
		t.Fatalf("default: %+v", *v)
	}

	f, v = newFlags()
	if err := f.Parse([]string{"--config", "none"}); err != nil {
		t.Fatal(err)
	}
	if v.Value() != "" || !v.IsNone() || !v.IsSet() || v.String() != "none" {
		t.Fatalf("none: %+v", *v)
	}

	f, v = newFlags()
	if err := f.Parse([]string{"--config=custom.yaml"}); err != nil {
		t.Fatal(err)
	}
	if v.Value() != "custom.yaml" || v.IsNone() {
		t.Fatalf("custom: %+v", *v)
	}

	f, _ = newFlags()
	if err := f.Parse([]string{"--config="}); err == nil || !strings.Contains(err.Error(), `"none"`) {
		t.Fatalf("empty values should be rejected: %v", err)
	}

	if !strings.Contains(f.Lookup("config").Usage, `pass "none" to disable`) {
		t.Fatal("usage should mention none")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestHumanizeError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		hint    string
	}{
		{
			name:    "permission denied",
			err:     &os.PathError{Op: "open", Path: "file.txt", Err: syscall.EACCES},
			message: "Permission denied: file.txt",
			hint:    "chmod",
		},
		{
			name:    "not found",
			err:     fmt.Errorf("reading config: %w", &os.PathError{Op: "open", Path: "config.yaml", Err: syscall.ENOENT}),
			message: "No such file or directory: config.yaml",
			hint:    "spelled correctly",
		},
		{
			name:    "connection refused",
			err:     &net.OpError{Op: "dial", Net: "tcp", Addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}},
			message: "Connection refused by 127.0.0.1:8080",
			hint:    "server is running",
		},
		{
			name:    "deadline",
			err:     fmt.Errorf("fetching: %w", context.DeadlineExceeded),
			message: "The operation timed out.",
			hint:    "--timeout",
		},
		{
			name:    "network timeout",
			err:     timeoutError{},
			message: "The operation timed out.",
			hint:    "--timeout",
		},
		{
			name:    "dns",
			err:     &net.DNSError{Err: "no such host", Name: "example.invalid", IsNotFound: true},
			message: "Unable to resolve example.invalid",
			hint:    "hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var userErr *UserError
			if !errors.As(HumanizeError(tt.err), &userErr) {
				t.Fatalf("expected a UserError, got %v", HumanizeError(tt.err))
			}
			if userErr.Message != tt.message {
				t.Errorf("message = %q, want %q", userErr.Message, tt.message)
			}
			if !strings.Contains(userErr.Hint, tt.hint) {
				t.Errorf("hint = %q, want it to mention %q", userErr.Hint, tt.hint)
			}
			if !errors.Is(userErr, tt.err) && userErr.Err != tt.err {
				t.Error("the original error should be kept")
			}
		})
	}

	plain := errors.New("out of lollipops")
	if HumanizeError(plain) != plain {
		t.Error("unknown errors should be returned unchanged")
	}

	usage := NewUsageError("bad flag")
	if HumanizeError(usage) != usage {
		t.Error("framework errors should be returned unchanged")
	}

	if HumanizeError(nil) != nil {
		t.Error("nil should stay nil")
	}
}

func TestExecuteHumanizesErrors(t *testing.T) {
	c, _, stderr := newExecuteCommand(nil)
	code := Execute(c, nil, func(map[string]Argument) error {
		_, err := os.Open(filepath.Join(t.TempDir(), "missing.txt"))
		return err
	})

	if code != ExitCodeError {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "No such file or directory") || !strings.Contains(stderr.String(), "spelled correctly") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestVersionCommand(t *testing.T) {
	newVersion := func(settings *Settings) (*VersionCommand, *testCommand) {
		inner, _, _ := newExecuteCommand(settings)
		return &VersionCommand{Meta: inner.Meta}, inner
	}

	c, _ := newVersion(nil)
	stdout := c.stdout.(interface{ String() string })
	if code := c.Run(nil); code != 0 || stdout.String() != "1.2.3\n" {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}

	c, _ = newVersion(nil)
	stdout = c.stdout.(interface{ String() string })
	if code := c.Run([]string{"--json"}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	var info BuildInfo
	if err := json.Unmarshal([]byte(stdout.String()), &info); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if info.Version != "1.2.3" || info.GoVersion == "" || info.OS == "" {
		t.Fatalf("info = %+v", info)
	}

	c, _ = newVersion(&Settings{Debug: true})
	stderr := c.stderr.(interface{ String() string })
	c.Run(nil)
	if !strings.Contains(stderr.String(), "Go version:") || !strings.Contains(stderr.String(), "Platform:") {
		t.Fatalf("debug output = %q", stderr.String())
	}
}
