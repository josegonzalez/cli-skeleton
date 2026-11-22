package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
)

// RunFunc runs a command once its flags and arguments are parsed.
type RunFunc func(arguments map[string]Argument) error

// Execute parses the flags and arguments of c, calls run with the parsed
// arguments and turns the result into an exit code. It replaces the
// parse-and-report boilerplate in a command's Run method:
//
//	func (c *EatCommand) Run(args []string) int {
//		return command.Execute(c, args, func(arguments map[string]command.Argument) error {
//			c.Ui.Output("Eating a lollipop")
//			return nil
//		})
//	}
//
// Errors are reported as follows:
//
//   - -h or --help prints the command help to stdout and exits 0.
//   - A UsageError, including invalid flags and arguments, prints the error
//     and a pointer to the help and exits with ExitCodeUsage.
//   - A UserError prints its message and then its hint.
//   - A MultiError prints its header and each error beneath it.
//   - context.Canceled, after an interrupt, exits with ExitCodeInterrupt.
//   - An InternalError or a panic prints a short message and a link to
//     report the problem. Details are only shown under --debug.
//   - Any other error prints its message and exits with ExitCodeError.
func Execute(c Command, args []string, run RunFunc) (code int) {
	m := c.meta()
	defer func() {
		if r := recover(); r != nil {
			code = m.reportError(c, panicError(r))
		}
	}()

	arguments, err := Parse(c, args)
	if err == nil {
		err = run(arguments)
	}
	return m.reportError(c, err)
}

// Parse parses the flags and arguments of c. It returns ErrHelp when help
// was requested, and a UsageError when the flags or arguments are invalid.
func Parse(c Command, args []string) (map[string]Argument, error) {
	flags := c.FlagSet()
	flags.Usage = func() {}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, &UsageError{Err: err, Hint: unknownFlagHint(c, flags, err)}
	}

	if err := c.meta().validateStandardFlags(); err != nil {
		return nil, err
	}

	definitions := c.Arguments()
	positional, err := c.meta().promptMissingArguments(definitions, flags.Args())
	if err != nil {
		return nil, err
	}

	arguments, err := ParseArguments(positional, definitions)
	if err != nil {
		usageErr := &UsageError{Err: err}
		if len(positional) == 0 && hasRequiredArgument(definitions) {
			usageErr.showUsage = true
		}
		return nil, usageErr
	}

	return arguments, nil
}

// hasRequiredArgument reports whether any argument is required.
func hasRequiredArgument(arguments []Argument) bool {
	for _, argument := range arguments {
		if !argument.Optional {
			return true
		}
	}
	return false
}

// unknownFlagHint suggests the closest flag when err is about an unknown
// long flag, followed by the usual pointer to the help.
func unknownFlagHint(c Command, flags *flag.FlagSet, err error) string {
	var notExist *flag.NotExistError
	if !errors.As(err, &notExist) || notExist.GetSpecifiedName() == "" {
		return ""
	}

	var names []string
	flags.VisitAll(func(f *flag.Flag) {
		if !f.Hidden {
			names = append(names, f.Name)
		}
	})

	question := didYouMean(Suggest(notExist.GetSpecifiedName(), names), func(name string) string {
		return "--" + name
	})
	if question == "" {
		return ""
	}
	return question + "\n" + CommandErrorText(c)
}

// panicError converts a recovered panic into an InternalError.
func panicError(r interface{}) *InternalError {
	err, ok := r.(error)
	if !ok {
		err = fmt.Errorf("%v", r)
	}
	return &InternalError{Err: err, Stack: string(debug.Stack())}
}

// helpText returns the help of c, preferring its own Help method.
func helpText(c Command) string {
	if h, ok := c.(interface{ Help() string }); ok {
		return h.Help()
	}
	return CommandHelp(c)
}

// reportError prints err and returns the matching exit code. c may be nil
// when the error did not come from a command.
func (m *Meta) reportError(c Command, err error) int {
	if err == nil {
		return ExitCodeOK
	}

	if errors.Is(err, ErrHelp) && c != nil {
		m.writeOut(helpText(c))
		return ExitCodeOK
	}

	if errors.Is(err, context.Canceled) {
		return ExitCodeInterrupt
	}

	err = HumanizeError(err)

	switch e := err.(type) {
	case *UsageError:
		return m.reportUsageError(c, e)
	case *UserError:
		return m.reportUserError(e)
	case *MultiError:
		return m.reportMultiError(e)
	case *InternalError:
		return m.reportInternalError(c, e)
	}

	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		return m.reportUsageError(c, usageErr)
	}

	var userErr *UserError
	if errors.As(err, &userErr) {
		return m.reportUserError(userErr)
	}

	var internalErr *InternalError
	if errors.As(err, &internalErr) {
		return m.reportInternalError(c, internalErr)
	}

	m.Ui.Error(err.Error())
	return ExitCodeError
}

func (m *Meta) reportUsageError(c Command, err *UsageError) int {
	m.Ui.Error(err.Error())

	if err.showUsage && c != nil {
		m.writeErr("\n" + conciseCommandHelp(c))
		return ExitCodeUsage
	}

	hint := err.Hint
	if hint == "" && c != nil {
		hint = CommandErrorText(c)
	}
	if hint != "" {
		m.Ui.Info(hint)
	}
	return ExitCodeUsage
}

func (m *Meta) reportUserError(err *UserError) int {
	m.Ui.Error(err.Message)
	if err.Err != nil {
		m.Debugf("%s: %v", err.Message, err.Err)
	}
	if err.Hint != "" {
		m.Ui.Info(err.Hint)
	}
	return err.exitCode()
}

func (m *Meta) reportMultiError(err *MultiError) int {
	m.Ui.Error(err.Header)
	for _, e := range err.Errors {
		m.Ui.Error("  " + e.Error())
	}
	return ExitCodeError
}

func (m *Meta) reportInternalError(c Command, err *InternalError) int {
	where := m.appName
	if c != nil {
		where = m.appName + " " + c.Name()
	}
	m.Ui.Error(fmt.Sprintf("An unexpected error occurred while running '%s'.", where))

	m.Debugf("%v", err.Err)
	if err.Stack != "" {
		m.Debugf("%s", err.Stack)
	}

	if link := m.issueReportURL(c, err); link != "" {
		m.Ui.Info("Please report it at " + link)
	}
	if !m.IsDebug() {
		m.Ui.Info("Run the command again with --debug for details.")
	}
	return ExitCodeError
}

// issueReportURL returns a link to open an issue pre-filled with the build
// information and the error, or an empty string when no issue URL is set.
func (m *Meta) issueReportURL(c Command, err *InternalError) string {
	if m.issueURL == "" {
		return ""
	}

	u, parseErr := url.Parse(m.issueURL)
	if parseErr != nil {
		return m.issueURL
	}

	commandLine := m.appName
	if c != nil {
		commandLine += " " + c.Name()
	}

	info := m.BuildInfo()
	var body strings.Builder
	fmt.Fprintf(&body, "Command: `%s`\n", commandLine)
	fmt.Fprintf(&body, "Version: %s\n", valueOr(info.Version, "unknown"))
	fmt.Fprintf(&body, "Go: %s\n", info.GoVersion)
	fmt.Fprintf(&body, "Platform: %s/%s\n", info.OS, info.Arch)
	if info.Commit != "" {
		fmt.Fprintf(&body, "Commit: %s\n", info.Commit)
	}
	fmt.Fprintf(&body, "\nError:\n```\n%s\n```\n", err.Err.Error())

	q := u.Query()
	q.Set("title", fmt.Sprintf("Unexpected error in `%s`", commandLine))
	q.Set("body", body.String())
	u.RawQuery = q.Encode()
	return u.String()
}

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// WithIssueURL sets the URL where users report bugs, such as
// `https://github.com/org/repo/issues/new`. Unexpected errors print a link
// to it, pre-filled with the build information and the error.
func WithIssueURL(issueURL string) Option {
	return func(o *mainOptions) {
		o.issueURL = issueURL
	}
}

// Debugf writes a debugging message. When <APP>_DEBUG_LOG names a file,
// the message is appended to that file. Otherwise it is written to stderr
// under --debug, and dropped without it.
func (m *Meta) Debugf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	if m.debugLog != nil {
		for _, line := range strings.Split(strings.TrimRight(message, "\n"), "\n") {
			fmt.Fprintf(m.debugLog, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
		}
		return
	}

	if !m.IsDebug() {
		return
	}

	w := m.stderr
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintln(w, strings.TrimRight(message, "\n"))
}

// openDebugLog opens the file named by <APP>_DEBUG_LOG for appending.
func openDebugLog(appName string) (io.WriteCloser, error) {
	name := AppEnv(appName, "DEBUG_LOG")
	if name == "" {
		return nil, nil
	}

	path := os.Getenv(name)
	if path == "" {
		return nil, nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s=%s: %w", name, path, err)
	}
	return f, nil
}
