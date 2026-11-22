package command

import (
	"io"
	"os"

	"github.com/mitchellh/cli"
	"github.com/rs/zerolog"
)

type ZerologUi struct {
	StderrLogger      zerolog.Logger
	StdoutLogger      zerolog.Logger
	OriginalFields    map[string]interface{}
	Ui                cli.Ui
	OutputIndentField bool

	human       bool
	stdout      io.Writer
	stderr      io.Writer
	stdoutColor bool
	stderrColor bool
}

func (u *ZerologUi) Ask(query string) (string, error) {
	return u.Ui.Ask(query)
}

func (u *ZerologUi) AskSecret(query string) (string, error) {
	return u.Ui.AskSecret(query)
}

// ColorEnabled implements ColorReporter.
func (u *ZerologUi) ColorEnabled() (bool, bool) {
	return u.stdoutColor, u.stderrColor
}

func (u *ZerologUi) Error(message string) {
	u.StderrLogger.Error().Msg(message)
}

// Info logs an informational message to stderr.
func (u *ZerologUi) Info(message string) {
	u.StderrLogger.Info().Msg(message)
}

// Output logs primary output to stdout.
func (u *ZerologUi) Output(message string) {
	if u.OutputIndentField {
		u.StdoutLogger.Info().Bool("_indent", false).Msg(message)
	} else {
		u.StdoutLogger.Info().Msg(message)
	}
}

func (u *ZerologUi) Warn(message string) {
	u.StderrLogger.Warn().Msg(message)
}

// LogHeader1 logs a top-level progress header to stderr.
func (u *ZerologUi) LogHeader1(message string) {
	u.StderrLogger.Info().Int("_header", 1).Msg(message)
}

// LogHeader2 logs a second-level progress header to stderr.
func (u *ZerologUi) LogHeader2(message string) {
	u.StderrLogger.Info().Int("_header", 2).Msg(message)
}

func (u *ZerologUi) Field(field string, value interface{}) *ZerologUi {
	return u.Fields(map[string]interface{}{field: value})
}

func (u *ZerologUi) Fields(newFields map[string]interface{}) *ZerologUi {
	fields := make(map[string]interface{}, len(u.OriginalFields)+len(newFields))
	for k, v := range u.OriginalFields {
		fields[k] = v
	}
	for k, v := range newFields {
		fields[k] = v
	}

	return newZerologUi(u.Ui, fields, zerologUiOptions{
		human:       u.human,
		stdout:      u.stdout,
		stderr:      u.stderr,
		stdoutColor: u.stdoutColor,
		stderrColor: u.stderrColor,
	})
}

// ZerologUiWithFields returns a ZerologUi that logs through zerolog's
// console writer. Color follows the wrapped UI when it implements
// ColorReporter, and ColorEnabled otherwise.
func ZerologUiWithFields(ui cli.Ui, fields map[string]interface{}) *ZerologUi {
	return newZerologUi(ui, fields, defaultZerologUiOptions(ui, false))
}

// HumanZerologUiWithFields returns a ZerologUi that logs through HumanWriter.
// Color follows the wrapped UI when it implements ColorReporter, and
// ColorEnabled otherwise.
func HumanZerologUiWithFields(ui cli.Ui, fields map[string]interface{}) *ZerologUi {
	return newZerologUi(ui, fields, defaultZerologUiOptions(ui, true))
}

type zerologUiOptions struct {
	human       bool
	stdout      io.Writer
	stderr      io.Writer
	stdoutColor bool
	stderrColor bool
}

func defaultZerologUiOptions(ui cli.Ui, human bool) zerologUiOptions {
	stdoutColor, stderrColor := uiColors(ui)
	return zerologUiOptions{
		human:       human,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		stdoutColor: stdoutColor,
		stderrColor: stderrColor,
	}
}

func newZerologUi(ui cli.Ui, fields map[string]interface{}, opts zerologUiOptions) *ZerologUi {
	var stdoutWriter, stderrWriter io.Writer
	if opts.human {
		stdoutWriter = NewHumanWriter(func(w *HumanWriter) {
			w.Out = opts.stdout
			w.NoColor = !opts.stdoutColor
		})
		stderrWriter = NewHumanWriter(func(w *HumanWriter) {
			w.Out = opts.stderr
			w.NoColor = !opts.stderrColor
		})
	} else {
		stdoutWriter = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = opts.stdout
			w.NoColor = !opts.stdoutColor
		})
		stderrWriter = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = opts.stderr
			w.NoColor = !opts.stderrColor
		})
	}

	return &ZerologUi{
		StderrLogger:      zerolog.New(stderrWriter).With().Fields(fields).Timestamp().Logger(),
		StdoutLogger:      zerolog.New(stdoutWriter).With().Fields(fields).Timestamp().Logger(),
		OriginalFields:    fields,
		Ui:                ui,
		OutputIndentField: opts.human,
		human:             opts.human,
		stdout:            opts.stdout,
		stderr:            opts.stderr,
		stdoutColor:       opts.stdoutColor,
		stderrColor:       opts.stderrColor,
	}
}

// uiColors returns the color settings of ui when it implements
// ColorReporter, and detects them from stdout and stderr otherwise.
func uiColors(ui cli.Ui) (bool, bool) {
	if r, ok := ui.(ColorReporter); ok {
		return r.ColorEnabled()
	}
	return ColorEnabled(os.Stdout, "", false), ColorEnabled(os.Stderr, "", false)
}

var _ cli.Ui = (*ZerologUi)(nil)
var _ ColorReporter = (*ZerologUi)(nil)
