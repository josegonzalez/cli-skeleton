package command

import (
	"context"
	"os"
	"strings"

	colorable "github.com/mattn/go-colorable"
	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

const (
	// EnvCLINoColor is an env var that toggles colored UI output.
	EnvCLINoColor = `NO_COLOR`
)

// NamedCommand is a interface to denote a commmand's name.
type NamedCommand interface {
	Name() string
}

type CommandFunc func(ctx context.Context, meta Meta) map[string]cli.CommandFactory

// Commands returns the mapping of CLI commands. The meta
// parameter lets you set meta options for all commands.
func Commands(ctx context.Context, metaPtr *Meta, commandsFunc CommandFunc) map[string]cli.CommandFactory {
	if metaPtr == nil {
		metaPtr = new(Meta)
	}

	meta := *metaPtr
	if meta.Ui == nil {
		meta.Ui = &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      colorable.NewColorableStdout(),
			ErrorWriter: colorable.NewColorableStderr(),
		}
	}
	meta.Context = ctx

	return commandsFunc(ctx, meta)
}

type Command interface {
	Name() string
	FlagSet() *flag.FlagSet
	Arguments() []Argument
	Synopsis() string
	Examples() map[string]string
}

func CommandHelp(c Command) string {
	appName := os.Getenv("CLI_APP_NAME")
	helpText := `
Usage: ` + appName + ` ` + c.Name() + ` ` + FlagString(c.FlagSet()) + ` ` + ArgumentAsString(c.Arguments()) + `

  ` + c.Synopsis()

	options := c.FlagSet().FlagUsages()
	if options != "" {
		helpText += `

Options:

` + options
	}

	arguments := ArgumentsString(c.Arguments())
	if arguments != "" {
		helpText += `
Arguments:

` + arguments
	}

	examples := ExampleString(c.Examples())
	if examples != "" {
		helpText += `
Examples:

` + examples
	}

	return strings.TrimSpace(helpText) + "\n"
}
