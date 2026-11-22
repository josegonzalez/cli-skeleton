package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
)

// Settings holds the framework-wide settings shared by every command.
//
// Main resolves them once, before the UI is built, from the environment and
// from a scan of the command-line arguments. Meta.FlagSet binds the
// framework flags to the same fields, so the values are identical before
// and after flags are parsed.
type Settings struct {
	// NoColor disables colored output (--no-color).
	NoColor bool

	// NoInput disables prompts (--no-input).
	NoInput bool

	// Quiet suppresses non-essential output (-q, --quiet).
	Quiet bool

	// Debug enables debugging output (-d, --debug, DEBUG or <APP>_DEBUG).
	Debug bool

	// Verbose streams detailed progress output (<APP>_VERBOSE). Debug
	// implies Verbose.
	Verbose bool

	// NoPager disables paging of long output (--no-pager).
	NoPager bool
}

// resolveSettings builds the settings for a run from the config files, the
// environment and the command-line arguments, each overriding the last.
func resolveSettings(appName string, args []string, flags *flag.FlagSet, cfg *configData) *Settings {
	s := &Settings{}
	cfg.applySettingsConfig(s)
	applyEnvSettings(s, appName)
	prescanSettings(s, args, flags)
	return s
}

// applyEnvSettings sets the settings that can be enabled through the
// environment.
func applyEnvSettings(s *Settings, appName string) {
	if envTruthy("DEBUG") || envTruthy(AppEnv(appName, "DEBUG")) {
		s.Debug = true
	}
	if envTruthy(AppEnv(appName, "VERBOSE")) {
		s.Verbose = true
	}
}

// envTruthy reports whether the env var is set to a value other than an
// empty string, `0` or `false`.
func envTruthy(name string) bool {
	if name == "" {
		return false
	}

	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false
	}

	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	return true
}

// settingFlags maps each framework flag to the setting it controls.
func settingFlags(s *Settings) map[string]*bool {
	return map[string]*bool{
		"no-color": &s.NoColor,
		"no-input": &s.NoInput,
		"quiet":    &s.Quiet,
		"debug":    &s.Debug,
		"no-pager": &s.NoPager,
	}
}

// prescanSettings applies framework flags found in args before the flags are
// parsed, so the UI can be configured before any command runs.
//
// It follows pflag syntax: long flags (with or without `=value`), short flags
// and clusters of short flags. Single-dash long flags such as `-no-color`
// are read the way pflag reads them, as a cluster of short flags. Scanning
// stops at `--`, which ends flag parsing.
//
// flags describes the known flags, so that values of value-taking flags are
// not mistaken for flags.
func prescanSettings(s *Settings, args []string, flags *flag.FlagSet) {
	settings := settingFlags(s)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return
		}

		if !strings.HasPrefix(arg, "-") || arg == "-" {
			continue
		}

		if strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(arg[2:], "=")
			if target, ok := settings[name]; ok {
				if !hasValue {
					*target = true
				} else if b, err := strconv.ParseBool(value); err == nil {
					*target = b
				}
				continue
			}

			if f := flags.Lookup(name); f != nil && takesValue(f) && !hasValue {
				i++
			}
			continue
		}

		for j := 1; j < len(arg); j++ {
			f := flags.ShorthandLookup(arg[j : j+1])
			if f == nil {
				continue
			}

			if takesValue(f) {
				if j == len(arg)-1 {
					i++
				}
				break
			}

			if target, ok := settings[f.Name]; ok {
				*target = true
			}
		}
	}
}

// takesValue reports whether a flag requires a value.
func takesValue(f *flag.Flag) bool {
	return f.NoOptDefVal == ""
}

// appEnvName returns the app-specific env var name for help text, or a
// placeholder when the app name is unknown.
func appEnvName(appName string, suffix string) string {
	if name := AppEnv(appName, suffix); name != "" {
		return name
	}
	return "<APP>_" + suffix
}

// bindBool binds a bool flag to p without resetting the current value of p,
// so values resolved before parsing survive the flag definition. The flag
// default shown in help stays `false`.
func bindBool(f *flag.FlagSet, p *bool, name string, shorthand string, usage string) {
	current := *p
	f.BoolVarP(p, name, shorthand, false, usage)
	*p = current
}

// addClientFlags adds the framework-global flags backed by s to f.
func addClientFlags(f *flag.FlagSet, s *Settings, appName string) {
	bindBool(f, &s.NoColor, "no-color", "", "disable colored output; NO_COLOR or "+appEnvName(appName, "NO_COLOR")+" may also be set")
	bindBool(f, &s.NoInput, "no-input", "", "never prompt for input; fail instead when input is required")
	bindBool(f, &s.Quiet, "quiet", "q", "only print essential output")
	bindBool(f, &s.Debug, "debug", "d", "print debugging output; DEBUG or "+appEnvName(appName, "DEBUG")+" may also be set")
	bindBool(f, &s.NoPager, "no-pager", "", "do not page long output")
}

// ErrNoInput is returned when a prompt is needed but input is disabled.
var ErrNoInput = errors.New("input required")

// NoInputError is returned by prompts when input cannot be read from the
// user. It wraps ErrNoInput.
type NoInputError struct {
	// Reason explains why the prompt was refused.
	Reason string
}

func (e *NoInputError) Error() string {
	return "input required, but " + e.Reason
}

func (e *NoInputError) Unwrap() error {
	return ErrNoInput
}

// settingsProvider is implemented by UIs that know the run settings.
type settingsProvider interface {
	uiSettings() *Settings
}

// settingsOf returns the settings known to ui, or nil.
func settingsOf(ui cli.Ui) *Settings {
	if p, ok := ui.(settingsProvider); ok {
		return p.uiSettings()
	}
	return nil
}

// settingsUi wraps a cli.Ui and applies the run settings to it. It drops info
// messages under --quiet, and refuses to prompt under --no-input or when
// stdin is not a terminal.
type settingsUi struct {
	cli.Ui

	settings   *Settings
	stdin      io.Reader
	stderr     io.Writer
	interrupts *interruptHandler
}

func newSettingsUi(ui cli.Ui, settings *Settings, stdin io.Reader) *settingsUi {
	return &settingsUi{Ui: ui, settings: settings, stdin: stdin}
}

func (u *settingsUi) uiSettings() *Settings {
	return u.settings
}

// ColorEnabled implements ColorReporter.
func (u *settingsUi) ColorEnabled() (bool, bool) {
	return uiColors(u.Ui)
}

// Unwrap returns the wrapped UI.
func (u *settingsUi) Unwrap() cli.Ui {
	return u.Ui
}

func (u *settingsUi) Info(message string) {
	if u.settings.Quiet {
		return
	}
	u.Ui.Info(message)
}

func (u *settingsUi) Ask(query string) (string, error) {
	if err := u.canPrompt(); err != nil {
		return "", err
	}
	defer u.whilePrompting()()
	return u.Ui.Ask(query)
}

func (u *settingsUi) AskSecret(query string) (string, error) {
	if err := u.canPrompt(); err != nil {
		return "", err
	}
	defer u.whilePrompting()()
	return u.Ui.AskSecret(query)
}

// whilePrompting makes an interrupt during a prompt exit right away, after
// restoring the terminal, since there is nothing to clean up while waiting
// for input. It returns a function that ends the prompt.
func (u *settingsUi) whilePrompting() func() {
	if u.interrupts == nil {
		return func() {}
	}

	restore := saveTerminalState(u.stdin)
	return u.interrupts.register(interruptHook{
		immediate: true,
		fn: func() {
			restore()
			if u.stderr != nil {
				fmt.Fprintln(u.stderr)
			}
		},
	})
}

// saveTerminalState captures the state of stdin when it is a terminal, and
// returns a function that restores it. Prompts for secrets turn off echo,
// which must be turned back on when the process exits mid-prompt.
func saveTerminalState(stdin io.Reader) func() {
	f, ok := stdin.(*os.File)
	if !ok || !IsTerminal(f) {
		return func() {}
	}

	state, err := term.GetState(int(f.Fd()))
	if err != nil {
		return func() {}
	}
	return func() {
		_ = term.Restore(int(f.Fd()), state)
	}
}

// canPrompt returns a NoInputError when the user cannot be prompted.
func (u *settingsUi) canPrompt() error {
	return checkPrompt(u.settings, u.stdin)
}

// checkPrompt returns a NoInputError when the user cannot be prompted.
func checkPrompt(settings *Settings, stdin io.Reader) error {
	if settings != nil && settings.NoInput {
		return &NoInputError{Reason: "--no-input was passed"}
	}

	if !isTerminalReader(stdin) {
		return &NoInputError{Reason: "stdin is not a terminal"}
	}

	return nil
}

var _ cli.Ui = (*settingsUi)(nil)
var _ ColorReporter = (*settingsUi)(nil)
