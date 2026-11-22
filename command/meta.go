package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const (
	// Constants for CLI identifier length
	shortId = 8
	fullId  = 36
)

// FlagSetFlags is an enum to define what flags are present in the
// default FlagSet returned by Meta.FlagSet.
type FlagSetFlags uint

const (
	FlagSetNone FlagSetFlags = 0

	// FlagSetClient adds the framework-global flags: --no-color,
	// --no-input, -q/--quiet, -d/--debug and --no-pager. Their values are
	// shared by every command and resolved before the UI is built.
	FlagSetClient FlagSetFlags = 1 << iota

	// FlagSetJSON adds --json, read with Meta.IsJSON.
	FlagSetJSON

	// FlagSetPlain adds --plain, read with Meta.IsPlain.
	FlagSetPlain

	// FlagSetForce adds -f/--force, read with Meta.IsForce.
	FlagSetForce

	// FlagSetDryRun adds -n/--dry-run, read with Meta.IsDryRun.
	FlagSetDryRun

	// FlagSetConfirm adds --confirm, used by Meta.ConfirmName.
	FlagSetConfirm

	// FlagSetTimeout adds --timeout, read with Meta.Timeout and
	// Meta.TimeoutContext.
	FlagSetTimeout

	// FlagSetOutput adds -o/--output, opened with Meta.OutputWriter.
	FlagSetOutput

	// FlagSetAll adds -a/--all, read with Meta.IsAll.
	FlagSetAll

	// FlagSetPort adds -p/--port, read with Meta.Port.
	FlagSetPort

	// FlagSetUser adds -u/--user, read with Meta.User.
	FlagSetUser

	// FlagSetDefault adds the framework-global flags and every standard flag.
	FlagSetDefault = FlagSetClient | FlagSetJSON | FlagSetPlain | FlagSetForce |
		FlagSetDryRun | FlagSetConfirm | FlagSetTimeout | FlagSetOutput |
		FlagSetAll | FlagSetPort | FlagSetUser
)

// Meta contains the meta-options and functionality that nearly
// every command inherits.
type Meta struct {
	Ui cli.Ui

	Context context.Context

	// The name of the cli tool
	appName string

	// The version of the cli tool
	version string

	// The settings shared by every command in the run
	settings *Settings

	// The interrupt handler of the run
	interrupts *interruptHandler

	// The URL where users report bugs
	issueURL string

	// Where debugging output is written under --debug
	stderr io.Writer

	// The file named by <APP>_DEBUG_LOG, when set
	debugLog io.Writer

	// Where help and other direct output is written
	stdout io.Writer

	// The process stdout and stderr, used to detect terminals
	stdoutFile *os.File
	stderrFile *os.File

	// The app description and examples shown in the top-level help
	description string
	examples    []Example

	// Links shown at the bottom of help output
	docsURL    string
	supportURL string

	// Commands hidden from the command list
	hiddenCommands []string

	// Explicit aliases, mapping each alias to its command
	aliases map[string]string

	// The values of the opt-in standard flags of the command
	flags commandFlags

	// The defaults of --timeout and --port
	defaultTimeout *time.Duration
	defaultPort    int

	// The flag groups the command added with FlagSet
	flagBits FlagSetFlags

	// The stdin of the run
	stdin io.Reader
}

// writeOut writes text to stdout, bypassing any log formatting of the UI.
func (m *Meta) writeOut(text string) {
	w := m.stdout
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprint(w, ensureNewline(text))
}

// writeErr writes text to stderr, bypassing any log formatting of the UI.
func (m *Meta) writeErr(text string) {
	w := m.stderr
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprint(w, ensureNewline(text))
}

func ensureNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

// meta returns the Meta. Commands satisfy the Command interface by
// embedding Meta.
func (m *Meta) meta() *Meta {
	return m
}

// AppName returns the name of the cli tool.
func (m *Meta) AppName() string {
	return m.appName
}

// Version returns the version of the cli tool.
func (m *Meta) Version() string {
	return m.version
}

// ensureSettings returns the run settings, creating empty ones when the Meta
// was not created by Main.
func (m *Meta) ensureSettings() *Settings {
	if m.settings == nil {
		m.settings = &Settings{}
	}
	return m.settings
}

// IsNoColor reports whether --no-color was passed.
func (m *Meta) IsNoColor() bool {
	return m.ensureSettings().NoColor
}

// IsNoInput reports whether --no-input was passed.
func (m *Meta) IsNoInput() bool {
	return m.ensureSettings().NoInput
}

// IsQuiet reports whether -q/--quiet was passed.
func (m *Meta) IsQuiet() bool {
	return m.ensureSettings().Quiet
}

// IsDebug reports whether debugging output is enabled through -d/--debug,
// DEBUG or <APP>_DEBUG.
func (m *Meta) IsDebug() bool {
	return m.ensureSettings().Debug
}

// IsVerbose reports whether verbose output is enabled through
// <APP>_VERBOSE or debug mode.
func (m *Meta) IsVerbose() bool {
	s := m.ensureSettings()
	return s.Verbose || s.Debug
}

// IsNoPager reports whether --no-pager was passed.
func (m *Meta) IsNoPager() bool {
	return m.ensureSettings().NoPager
}

// FlagSet returns a FlagSet with the common flags that every
// command implements. The exact behavior of FlagSet can be configured
// using the flags as the second parameter, for example to disable
// server settings on the commands that don't talk to a server.
func (m *Meta) FlagSet(n string, fs FlagSetFlags) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)

	if fs&FlagSetClient != 0 {
		addClientFlags(f, m.ensureSettings(), m.appName)
	}
	m.addStandardFlags(f, fs)
	m.flagBits |= fs

	f.SetOutput(&uiErrorWriter{ui: m.Ui})

	return f
}

// AutocompleteFlags returns a set of flag completions for the given flag set.
func (m *Meta) AutocompleteFlags(fs FlagSetFlags) complete.Flags {
	flags := standardFlagCompletions(fs)
	if fs&FlagSetClient != 0 {
		for name, predictor := range map[string]complete.Predictor{
			"--no-color": complete.PredictNothing,
			"--no-input": complete.PredictNothing,
			"--quiet":    complete.PredictNothing,
			"-q":         complete.PredictNothing,
			"--debug":    complete.PredictNothing,
			"-d":         complete.PredictNothing,
			"--no-pager": complete.PredictNothing,
		} {
			flags[name] = predictor
		}
	}

	if len(flags) == 0 {
		return nil
	}
	return flags
}

func (m *Meta) Colorize() *colorstring.Colorize {
	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: !ColorEnabled(os.Stdout, m.appName, m.IsNoColor()),
		Reset:   true,
	}
}

type GlobalFlagCommand interface {
	GlobalFlags(*flag.FlagSet)
}

// funcVar is a type of flag that accepts a function that is the string given
// by the user.
type funcVar func(s string) error

func (f funcVar) Set(s string) error { return f(s) }
func (f funcVar) String() string     { return "" }
func (f funcVar) IsBoolFlag() bool   { return false }
