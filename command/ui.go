package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	colorable "github.com/mattn/go-colorable"
	"github.com/mitchellh/cli"
	"golang.org/x/term"
)

// ColorReporter is implemented by a cli.Ui that knows whether its output
// streams are colored. Wrapping UIs such as ZerologUi use it to match the
// color settings of the UI they wrap.
type ColorReporter interface {
	ColorEnabled() (stdout bool, stderr bool)
}

// TerminalUi is a cli.Ui that writes to a pair of streams and decides
// whether to use color for each stream separately. Primary output goes to
// Writer, while info messages, warnings, errors and prompts go to
// ErrorWriter. Errors are red and warnings are yellow. Output and info
// messages are never colored.
//
// It is safe for concurrent use.
type TerminalUi struct {
	// Reader is where answers to prompts are read from.
	Reader io.Reader

	// Writer receives primary output.
	Writer io.Writer

	// ErrorWriter receives info messages, warnings, errors and prompts.
	ErrorWriter io.Writer

	// OutputColor enables color on Writer.
	OutputColor bool

	// ErrorColor enables color on ErrorWriter.
	ErrorColor bool

	mu     sync.Mutex
	reader *bufio.Reader
}

// NewTerminalUi returns a TerminalUi attached to stdin, stdout and stderr.
// Color is enabled for each stream according to ColorEnabled.
func NewTerminalUi(appName string, noColor bool) *TerminalUi {
	return &TerminalUi{
		Reader:      os.Stdin,
		Writer:      colorable.NewColorableStdout(),
		ErrorWriter: colorable.NewColorableStderr(),
		OutputColor: ColorEnabled(os.Stdout, appName, noColor),
		ErrorColor:  ColorEnabled(os.Stderr, appName, noColor),
	}
}

// ColorEnabled implements ColorReporter.
func (u *TerminalUi) ColorEnabled() (bool, bool) {
	return u.OutputColor, u.ErrorColor
}

// Ask asks the user for input using the given query. The response is
// returned without the trailing newline.
func (u *TerminalUi) Ask(query string) (string, error) {
	return u.ask(query, false)
}

// AskSecret asks the user for input using the given query, without echoing
// the response when the reader is a terminal.
func (u *TerminalUi) AskSecret(query string) (string, error) {
	return u.ask(query, true)
}

func (u *TerminalUi) ask(query string, secret bool) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if query != "" {
		if _, err := fmt.Fprint(u.promptWriter(), query+" "); err != nil {
			return "", err
		}
	}

	if secret {
		if f, ok := u.Reader.(*os.File); ok && IsTerminal(f) {
			line, err := term.ReadPassword(int(f.Fd()))
			fmt.Fprintln(u.promptWriter())
			if err != nil {
				return "", err
			}
			return string(line), nil
		}
	}

	if u.reader == nil {
		if u.Reader == nil {
			return "", errors.New("no input reader configured")
		}
		u.reader = bufio.NewReader(u.Reader)
	}

	line, err := u.reader.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", err
	}

	return strings.TrimRight(line, "\r\n"), nil
}

// promptWriter returns the stream prompts are written to. Prompts are
// messaging rather than output, so they go to stderr and never end up in a
// pipe.
func (u *TerminalUi) promptWriter() io.Writer {
	return u.ErrorWriter
}

// Output writes a message to Writer.
func (u *TerminalUi) Output(message string) {
	u.write(u.Writer, message)
}

// Info writes an informational message to ErrorWriter. Informational
// messages are not primary output, so they stay out of pipes.
func (u *TerminalUi) Info(message string) {
	u.write(u.ErrorWriter, message)
}

// Error writes an error message to ErrorWriter, in red when ErrorColor is set.
func (u *TerminalUi) Error(message string) {
	u.write(u.ErrorWriter, colorize(message, colorRed, !u.ErrorColor))
}

// Warn writes a warning message to ErrorWriter, in yellow when ErrorColor is set.
func (u *TerminalUi) Warn(message string) {
	u.write(u.ErrorWriter, colorize(message, colorYellow, !u.ErrorColor))
}

func (u *TerminalUi) write(w io.Writer, message string) {
	if w == nil {
		return
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Fprintln(w, message)
}

var _ cli.Ui = (*TerminalUi)(nil)
var _ ColorReporter = (*TerminalUi)(nil)
