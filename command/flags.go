package command

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

// DefaultTimeout is the default value of --timeout.
const DefaultTimeout = 30 * time.Second

// commandFlags holds the values of the opt-in standard flags of a command.
type commandFlags struct {
	json    bool
	plain   bool
	force   bool
	dryRun  bool
	confirm string
	timeout time.Duration
	output  string
	all     bool
	port    int
	user    string
}

// WithDefaultTimeout sets the default of --timeout for commands that use
// FlagSetTimeout. It defaults to DefaultTimeout.
func WithDefaultTimeout(d time.Duration) Option {
	return func(o *mainOptions) {
		o.defaultTimeout = &d
	}
}

// WithDefaultPort sets the default of -p/--port for commands that use
// FlagSetPort.
func WithDefaultPort(port int) Option {
	return func(o *mainOptions) {
		o.defaultPort = port
	}
}

// addStandardFlags adds the opt-in standard flags selected by fs.
func (m *Meta) addStandardFlags(f *flag.FlagSet, fs FlagSetFlags) {
	v := &m.flags

	if fs&FlagSetJSON != 0 {
		f.BoolVar(&v.json, "json", false, "output JSON")
	}
	if fs&FlagSetPlain != 0 {
		f.BoolVar(&v.plain, "plain", false, "output plain, tab-separated text with one record per line")
	}
	if fs&FlagSetForce != 0 {
		f.BoolVarP(&v.force, "force", "f", false, "skip confirmation prompts")
	}
	if fs&FlagSetDryRun != 0 {
		f.BoolVarP(&v.dryRun, "dry-run", "n", false, "show what would change without changing anything")
	}
	if fs&FlagSetConfirm != 0 {
		f.StringVar(&v.confirm, "confirm", "", "confirm a dangerous action by passing the name of the resource")
	}
	if fs&FlagSetTimeout != 0 {
		timeout := DefaultTimeout
		if m.defaultTimeout != nil {
			timeout = *m.defaultTimeout
		}
		f.DurationVar(&v.timeout, "timeout", timeout, "maximum time to wait for network operations")
	}
	if fs&FlagSetOutput != 0 {
		f.StringVarP(&v.output, "output", "o", "", "write output to a file instead of stdout; - means stdout")
	}
	if fs&FlagSetAll != 0 {
		f.BoolVarP(&v.all, "all", "a", false, "include all items")
	}
	if fs&FlagSetPort != 0 {
		f.IntVarP(&v.port, "port", "p", m.defaultPort, "port to connect to")
	}
	if fs&FlagSetUser != 0 {
		f.StringVarP(&v.user, "user", "u", "", "user to act as")
	}
}

// standardFlagCompletions returns completions for the opt-in standard flags
// selected by fs.
func standardFlagCompletions(fs FlagSetFlags) complete.Flags {
	flags := complete.Flags{}
	add := func(bit FlagSetFlags, predictor complete.Predictor, names ...string) {
		if fs&bit != 0 {
			for _, name := range names {
				flags[name] = predictor
			}
		}
	}

	add(FlagSetJSON, complete.PredictNothing, "--json")
	add(FlagSetPlain, complete.PredictNothing, "--plain")
	add(FlagSetForce, complete.PredictNothing, "--force", "-f")
	add(FlagSetDryRun, complete.PredictNothing, "--dry-run", "-n")
	add(FlagSetConfirm, complete.PredictAnything, "--confirm")
	add(FlagSetTimeout, complete.PredictAnything, "--timeout")
	add(FlagSetOutput, complete.PredictFiles("*"), "--output", "-o")
	add(FlagSetAll, complete.PredictNothing, "--all", "-a")
	add(FlagSetPort, complete.PredictAnything, "--port", "-p")
	add(FlagSetUser, complete.PredictAnything, "--user", "-u")
	return flags
}

// validateStandardFlags checks combinations of standard flags.
func (m *Meta) validateStandardFlags() error {
	if m.flags.json && m.flags.plain {
		return &UsageError{Err: errors.New("--json and --plain cannot be used together")}
	}
	return nil
}

// IsJSON reports whether --json was passed.
func (m *Meta) IsJSON() bool {
	return m.flags.json
}

// IsPlain reports whether --plain was passed.
func (m *Meta) IsPlain() bool {
	return m.flags.plain
}

// IsForce reports whether -f/--force was passed.
func (m *Meta) IsForce() bool {
	return m.flags.force
}

// IsDryRun reports whether -n/--dry-run was passed.
func (m *Meta) IsDryRun() bool {
	return m.flags.dryRun
}

// IsAll reports whether -a/--all was passed.
func (m *Meta) IsAll() bool {
	return m.flags.all
}

// Port returns the value of -p/--port.
func (m *Meta) Port() int {
	return m.flags.port
}

// User returns the value of -u/--user.
func (m *Meta) User() string {
	return m.flags.user
}

// Timeout returns the value of --timeout. Zero means no timeout.
func (m *Meta) Timeout() time.Duration {
	return m.flags.timeout
}

// TimeoutContext returns a context that is cancelled on interrupt or when
// --timeout runs out. A zero timeout only cancels on interrupt.
func (m *Meta) TimeoutContext() (context.Context, context.CancelFunc) {
	ctx := m.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if m.flags.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, m.flags.timeout)
}

// OutputPath returns the value of -o/--output. An empty string or `-`
// means stdout.
func (m *Meta) OutputPath() string {
	return m.flags.output
}

// OutputWriter opens the destination named by -o/--output, or stdout when
// it is empty or `-`. The caller must close it; closing stdout is a no-op.
func (m *Meta) OutputWriter() (io.WriteCloser, error) {
	if m.flags.output == "" || m.flags.output == "-" {
		w := m.stdout
		if w == nil {
			return OpenOutput("-")
		}
		return nopWriteCloser{w}, nil
	}
	return OpenOutput(m.flags.output)
}
