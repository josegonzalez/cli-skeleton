package command

import (
	"context"
	"strings"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

const (
	// EnvCLINoColor is an env var that toggles colored UI output.
	EnvCLINoColor = `NO_COLOR`
)

// NamedCommand is a interface to denote a commmand's name. AppName is
// provided by embedding Meta.
type NamedCommand interface {
	AppName() string
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
		meta.Ui = NewTerminalUi(meta.appName, meta.IsNoColor())
	}
	meta.Context = ctx

	return commandsFunc(ctx, meta)
}

// Command is implemented by every cli-skeleton command. Commands satisfy it
// by embedding Meta, which provides AppName and the unexported accessors
// used by Execute.
type Command interface {
	meta() *Meta

	AppName() string
	Name() string
	FlagSet() *flag.FlagSet
	Arguments() []Argument
	Synopsis() string
	Examples() map[string]string
}

func CommandHelp(c Command) string {
	helpText := `
Usage: ` + c.AppName() + ` ` + c.Name() + ` ` + FlagString(c.FlagSet()) + ` ` + ArgumentAsString(c.Arguments()) + `

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
