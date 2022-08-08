package command

import (
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
}

func (u *ZerologUi) Ask(query string) (string, error) {
	return u.Ui.Ask(query)
}

func (u *ZerologUi) AskSecret(query string) (string, error) {
	return u.Ui.AskSecret(query)
}

func (u *ZerologUi) Error(message string) {
	u.StderrLogger.Error().Msg(message)
}

func (u *ZerologUi) Info(message string) {
	u.StdoutLogger.Info().Msg(message)
}

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

func (u *ZerologUi) LogHeader1(message string) {
	u.StdoutLogger.Info().Int("_header", 1).Msg(message)
}

func (u *ZerologUi) LogHeader2(message string) {
	u.StdoutLogger.Info().Int("_header", 2).Msg(message)
}

func (u *ZerologUi) Field(field string, value interface{}) *ZerologUi {
	fields := make(map[string]interface{}, len(u.OriginalFields)+1)
	for k, v := range u.OriginalFields {
		fields[k] = v
	}

	fields[field] = value
	return ZerologUiWithFields(u.Ui, fields)
}

func (u *ZerologUi) Fields(newFields map[string]interface{}) *ZerologUi {
	fields := make(map[string]interface{}, len(u.OriginalFields)+len(newFields))
	for k, v := range u.OriginalFields {
		fields[k] = v
	}
	for k, v := range newFields {
		fields[k] = v
	}

	return ZerologUiWithFields(u.Ui, fields)
}

func ZerologUiWithFields(ui cli.Ui, fields map[string]interface{}) *ZerologUi {
	stderrWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
	})
	stdoutWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stdout
	})
	return &ZerologUi{
		StderrLogger:      zerolog.New(stderrWriter).With().Fields(fields).Timestamp().Logger(),
		StdoutLogger:      zerolog.New(stdoutWriter).With().Fields(fields).Timestamp().Logger(),
		OriginalFields:    fields,
		Ui:                ui,
		OutputIndentField: false,
	}
}

func HumanZerologUiWithFields(ui cli.Ui, fields map[string]interface{}) *ZerologUi {
	stderrWriter := NewHumanWriter(func(w *HumanWriter) {
		w.Out = os.Stderr
	})
	stdoutWriter := NewHumanWriter(func(w *HumanWriter) {
		w.Out = os.Stdout
	})
	return &ZerologUi{
		StderrLogger:      zerolog.New(stderrWriter).With().Fields(fields).Timestamp().Logger(),
		StdoutLogger:      zerolog.New(stdoutWriter).With().Fields(fields).Timestamp().Logger(),
		OriginalFields:    fields,
		Ui:                ui,
		OutputIndentField: true,
	}
}
