package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	colorable "github.com/mattn/go-colorable"
	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// Exit codes returned by Main and Execute.
const (
	// ExitCodeOK is returned when a command succeeds.
	ExitCodeOK = 0

	// ExitCodeError is returned when a command fails.
	ExitCodeError = 1

	// ExitCodeUsage is returned when a command is called with invalid
	// flags or arguments.
	ExitCodeUsage = 2

	// ExitCodeUnknownCommand is returned when the requested command does not
	// exist.
	ExitCodeUnknownCommand = 127

	// ExitCodeInterrupt is returned when a command is interrupted by a
	// signal.
	ExitCodeInterrupt = 130
)

// Option configures Main.
type Option func(*mainOptions)

type mainOptions struct {
	ui          func(cli.Ui) cli.Ui
	globalFlags func(*flag.FlagSet)
	gracePeriod *time.Duration
	exit        func(int)
	issueURL    string
	description string
	examples    []Example
	docsURL     string
	supportURL  string

	defaultTimeout *time.Duration
	defaultPort    int
	tailLines      *int
	config         bool
	aliases        map[string]string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// WithUi replaces the UI used by commands. The function receives the
// default UI, which already applies the run settings, and returns the UI
// to use, such as a ZerologUi wrapping it.
func WithUi(fn func(cli.Ui) cli.Ui) Option {
	return func(o *mainOptions) {
		o.ui = fn
	}
}

// WithGlobalFlags registers the app-global flags defined by fn. Main uses
// them to accept those flags before the subcommand as well as after it.
// Commands still need to add the flags to their own FlagSet.
func WithGlobalFlags(fn func(*flag.FlagSet)) Option {
	return func(o *mainOptions) {
		o.globalFlags = fn
	}
}

// WithDescription sets the description shown at the top of the app help.
func WithDescription(description string) Option {
	return func(o *mainOptions) {
		o.description = description
	}
}

// WithExamples sets the examples shown in the app help. The first two are
// also shown when the app is run without arguments.
func WithExamples(examples ...Example) Option {
	return func(o *mainOptions) {
		o.examples = examples
	}
}

// WithDocsURL sets the base URL of the web docs. App help links to it, and
// command help links to `<docsURL>/<command-path>`, such as
// `<docsURL>/lollipop-eat` for `lollipop eat`.
func WithDocsURL(docsURL string) Option {
	return func(o *mainOptions) {
		o.docsURL = docsURL
	}
}

// WithSupportURL sets where users can get help or report issues. It is shown
// at the bottom of help output.
func WithSupportURL(supportURL string) Option {
	return func(o *mainOptions) {
		o.supportURL = supportURL
	}
}

// withExit replaces os.Exit for forced exits. It is used by tests.
func withExit(fn func(int)) Option {
	return func(o *mainOptions) {
		o.exit = fn
	}
}

// withStreams replaces stdin, stdout and stderr. It is used by tests.
func withStreams(stdin io.Reader, stdout io.Writer, stderr io.Writer) Option {
	return func(o *mainOptions) {
		o.stdin = stdin
		o.stdout = stdout
		o.stderr = stderr
	}
}

// Main runs the cli tool and returns its exit code.
//
//	func main() {
//		os.Exit(command.Main(context.Background(), AppName, Version, os.Args[1:], Commands))
//	}
//
// It resolves the framework settings from the environment and args, builds
// the UI, moves framework and app-global flags placed before the subcommand
// to after it, and runs the matching command.
//
// The first SIGINT or SIGTERM prints a message and cancels Meta.Context, so
// commands should stop when the context is done. A second signal, or the
// grace period running out, exits with ExitCodeInterrupt.
func Main(ctx context.Context, appName string, version string, args []string, commandsFunc CommandFunc, opts ...Option) int {
	o := &mainOptions{}
	for _, opt := range opts {
		opt(o)
	}

	st := o.streams()

	var cfg *configData
	if o.config {
		var err error
		cfg, err = loadConfig(appName)
		if err != nil {
			fmt.Fprintln(st.err, err.Error())
			return ExitCodeError
		}
	}

	knownFlags := globalFlagSet(o.globalFlags)
	settings := resolveSettings(appName, args, knownFlags, cfg)

	stdin, stdout, stderr := st.in, st.out, st.err
	base := &TerminalUi{
		Reader:      stdin,
		Writer:      stdout,
		ErrorWriter: stderr,
		OutputColor: colorEnabled(IsTerminal(st.outFile), appName, settings.NoColor),
		ErrorColor:  colorEnabled(IsTerminal(st.errFile), appName, settings.NoColor),
	}

	inner := newSettingsUi(base, settings, stdin)
	inner.stderr = stderr
	var ui cli.Ui = inner
	if o.ui != nil {
		ui = o.ui(ui)
		if _, ok := ui.(settingsProvider); !ok {
			ui = newSettingsUi(ui, settings, stdin)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gracePeriod := DefaultGracePeriod
	if o.gracePeriod != nil {
		gracePeriod = *o.gracePeriod
	}
	exit := o.exit
	if exit == nil {
		exit = os.Exit
	}
	interrupts := newInterruptHandler(cancel, base.Warn, gracePeriod, exit)
	interrupts.start()
	defer interrupts.stop()
	inner.interrupts = interrupts

	meta := &Meta{
		Ui:         ui,
		Context:    ctx,
		appName:    appName,
		version:    version,
		settings:   settings,
		interrupts: interrupts,
		issueURL:    o.issueURL,
		stderr:      stderr,
		stdout:      stdout,
		stdoutFile:  st.outFile,
		stderrFile:  st.errFile,
		description: o.description,
		examples:    o.examples,
		docsURL:     o.docsURL,
		supportURL:  o.supportURL,

		defaultTimeout: o.defaultTimeout,
		defaultPort:    o.defaultPort,
		stdin:          stdin,
		tailLines:      o.tailLines,
		progressState:  &progressState{},
		config:         cfg,
		registry:       &commandRegistry{},
	}
	defer meta.progressState.close()

	debugLog, err := openDebugLog(appName)
	if err != nil {
		base.Warn(err.Error())
	}
	if debugLog != nil {
		defer debugLog.Close()
		meta.debugLog = debugLog
	}

	commands := Commands(ctx, meta, commandsFunc)
	addBuiltinCommands(meta, commands)
	hideDeprecatedCommands(meta, commands)
	addAliases(meta, commands, o.aliases)

	if isCompletionRequest() {
		addParentCommands(meta, commands)
		meta.registry.commands = commands
		c := &cli.CLI{
			Name:           appName,
			Version:        version,
			Args:           args,
			Commands:       commands,
			HiddenCommands: meta.hiddenCommands(),
			Autocomplete:   true,
			HelpWriter:     stdout,
			ErrorWriter:    stderr,
		}
		return runCLI(c, meta)
	}

	args = rewriteAutocompleteArgs(args)
	args = rewriteHelpArgs(moveGlobalFlags(args, knownFlags, commands), commands)
	if len(args) == 0 {
		meta.writeErr(meta.topLevelHelp(meta.visibleCommands(commands), true))
		return ExitCodeUsage
	}

	var code int
	args, code, ok := meta.resolveUnknownCommand(args, commands)
	if !ok {
		return code
	}
	addParentCommands(meta, commands)
	meta.registry.commands = commands

	help := &helpWriter{meta: meta}
	c := &cli.CLI{
		Name:           appName,
		Version:        version,
		Args:           args,
		Commands:       commands,
		HiddenCommands: meta.hiddenCommands(),
		HelpFunc: func(commands map[string]cli.CommandFactory) string {
			visible := map[string]cli.CommandFactory{}
			for name, factory := range commands {
				if !meta.isHidden(name) {
					visible[name] = factory
				}
			}
			return strings.TrimRight(meta.topLevelHelp(visible, false), "\n")
		},
		HelpWriter:  help,
		ErrorWriter: stderr,
	}

	code = runCLI(c, meta)
	help.flush()
	return code
}

// addBuiltinCommands registers the commands provided by the framework,
// unless the app defines commands with the same names.
func addBuiltinCommands(meta *Meta, commands map[string]cli.CommandFactory) {
	if _, ok := commands["examples"]; !ok {
		commands["examples"] = func() (cli.Command, error) {
			return &examplesCommand{Meta: *meta, commands: commands}, nil
		}
	}

	if _, ok := commands["docs"]; !ok {
		commands["docs"] = func() (cli.Command, error) {
			return &docsCommand{Meta: *meta, commands: commands}, nil
		}
		if meta.registry != nil {
			meta.registry.hidden = append(meta.registry.hidden, "docs")
		}
	}

	if _, ok := commands["completion"]; !ok {
		commands["completion"] = func() (cli.Command, error) {
			return &completionCommand{Meta: *meta}, nil
		}
	}

	if _, ok := commands["config"]; !ok && meta.config != nil {
		commands["config"] = func() (cli.Command, error) {
			return &configCommand{Meta: *meta, commands: commands}, nil
		}
	}
}

// addParentCommands registers a parent command for nested commands without
// one, such as `lollipop` for `lollipop eat`.
func addParentCommands(meta *Meta, commands map[string]cli.CommandFactory) {
	var parents []string
	for name := range commands {
		words := strings.Fields(name)
		for i := 1; i < len(words); i++ {
			parent := strings.Join(words[:i], " ")
			if _, ok := commands[parent]; !ok {
				parents = append(parents, parent)
			}
		}
	}

	for _, parent := range parents {
		parent := parent
		commands[parent] = func() (cli.Command, error) {
			return &parentCommand{Meta: *meta, name: parent}, nil
		}
	}
}

// resolveUnknownCommand reports unknown commands. When there is a single
// close match and the user can be prompted, it asks whether to run it
// instead, and never runs it without a yes. It returns the args to run, or
// false with the exit code when the command is unknown.
func (m *Meta) resolveUnknownCommand(args []string, commands map[string]cli.CommandFactory) ([]string, int, bool) {
	for attempt := 0; attempt < len(args); attempt++ {
		word, candidates := unknownCommand(args, commands)
		if word == "" {
			return args, ExitCodeOK, true
		}

		candidates = m.visibleNames(candidates)
		suggestions := Suggest(word, candidates)
		if len(suggestions) != 1 || !m.CanPrompt() {
			return args, m.reportUnknownCommand(word, candidates), false
		}

		m.Ui.Error(fmt.Sprintf("Unknown command %q for %q.", word, m.appName))
		answer, err := m.Ui.Ask(fmt.Sprintf("Did you mean %q? [y/N]", suggestions[0]))
		if err != nil || !isYes(answer) {
			m.Ui.Info(fmt.Sprintf("Run '%s --help' for a list of commands.", m.appName))
			return args, ExitCodeUnknownCommand, false
		}

		args = replaceFirst(args, word, suggestions[0])
	}
	return args, ExitCodeOK, true
}

// isYes reports whether answer is y or yes.
func isYes(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

// replaceFirst returns a copy of args with the first occurrence of old
// replaced by replacement.
func replaceFirst(args []string, old string, replacement string) []string {
	result := append([]string{}, args...)
	for i, arg := range result {
		if arg == old {
			result[i] = replacement
			break
		}
	}
	return result
}

// reportUnknownCommand prints an unknown command error with suggestions.
func (m *Meta) reportUnknownCommand(word string, candidates []string) int {
	m.Ui.Error(fmt.Sprintf("Unknown command %q for %q.", word, m.appName))
	if question := didYouMean(Suggest(word, candidates), strconv.Quote); question != "" {
		m.Ui.Info(question)
	}
	m.Ui.Info(fmt.Sprintf("Run '%s --help' for a list of commands.", m.appName))
	return ExitCodeUnknownCommand
}

// runCLI runs c, reporting errors and panics that escape the command.
func runCLI(c *cli.CLI, meta *Meta) (code int) {
	defer func() {
		if r := recover(); r != nil {
			code = meta.reportError(nil, panicError(r))
		}
	}()

	exitCode, err := c.Run()
	if err != nil {
		meta.Ui.Error(fmt.Sprintf("Error executing CLI: %s", err.Error()))
		return ExitCodeError
	}

	return exitCode
}

// runStreams holds the streams of a run. outFile and errFile are the
// process files behind out and err, used for terminal detection; they are
// nil when the streams were replaced.
type runStreams struct {
	in      io.Reader
	out     io.Writer
	err     io.Writer
	outFile *os.File
	errFile *os.File
}

// streams returns the configured streams, defaulting to the process streams.
func (o *mainOptions) streams() runStreams {
	st := runStreams{in: o.stdin, out: o.stdout, err: o.stderr}
	if st.in == nil {
		st.in = os.Stdin
	}
	if st.out == nil {
		st.out = colorable.NewColorableStdout()
		st.outFile = os.Stdout
	}
	if st.err == nil {
		st.err = colorable.NewColorableStderr()
		st.errFile = os.Stderr
	}
	return st
}

// globalFlagSet returns a FlagSet holding the framework-global flags and the
// app-global flags. It describes the flags Main may move and pre-scan, and
// is never parsed.
func globalFlagSet(globalFlags func(*flag.FlagSet)) *flag.FlagSet {
	f := flag.NewFlagSet("global", flag.ContinueOnError)
	addClientFlags(f, &Settings{}, "")
	if globalFlags != nil {
		globalFlags(f)
	}
	return f
}

// moveGlobalFlags moves known flags placed before the subcommand to after
// it, so `app --no-color eat` behaves like `app eat --no-color`. Flags that
// are not known, such as --help and --version, stop the move and are left
// in place.
//
// When no subcommand follows the moved flags, they are dropped, since their
// settings have already been applied.
func moveGlobalFlags(args []string, known *flag.FlagSet, commands map[string]cli.CommandFactory) []string {
	i := 0
	var moved []string
	for i < len(args) {
		n := matchKnownFlag(args, i, known)
		if n == 0 {
			break
		}
		moved = append(moved, args[i:i+n]...)
		i += n
	}

	if len(moved) == 0 {
		return args
	}

	rest := args[i:]
	if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
		return append([]string{}, rest...)
	}

	end := commandPathLength(rest, commands)
	result := make([]string, 0, len(args))
	result = append(result, rest[:end]...)
	result = append(result, moved...)
	result = append(result, rest[end:]...)
	return result
}

// matchKnownFlag returns the number of args taken by the known flag at
// args[i], or 0 when args[i] is not a known flag.
func matchKnownFlag(args []string, i int, known *flag.FlagSet) int {
	arg := args[i]
	if arg == "--" || arg == "-" || !strings.HasPrefix(arg, "-") {
		return 0
	}

	if strings.HasPrefix(arg, "--") {
		name, _, hasValue := strings.Cut(arg[2:], "=")
		f := known.Lookup(name)
		if f == nil {
			return 0
		}
		if takesValue(f) && !hasValue {
			if i+1 >= len(args) {
				return 1
			}
			return 2
		}
		return 1
	}

	for j := 1; j < len(arg); j++ {
		f := known.ShorthandLookup(arg[j : j+1])
		if f == nil {
			return 0
		}
		if takesValue(f) {
			if j == len(arg)-1 && i+1 < len(args) {
				return 2
			}
			return 1
		}
	}
	return 1
}

// commandPathLength returns how many leading args name the subcommand,
// matching nested commands such as `lollipop eat`. When nothing matches,
// the first arg is treated as the subcommand.
func commandPathLength(args []string, commands map[string]cli.CommandFactory) int {
	longest := 1
	for n := 1; n <= len(args); n++ {
		if strings.HasPrefix(args[n-1], "-") {
			break
		}
		if _, ok := commands[strings.Join(args[:n], " ")]; ok {
			longest = n
		}
	}
	return longest
}
