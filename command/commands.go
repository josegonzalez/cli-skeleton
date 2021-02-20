package command

import (
	"os"

	"github.com/mattn/go-colorable"
	"github.com/mitchellh/cli"
)

const (
	// EnvCLINoColor is an env var that toggles colored UI output.
	EnvCLINoColor = `NO_COLOR`
)

// NamedCommand is a interface to denote a commmand's name.
type NamedCommand interface {
	Name() string
}

type CommandFunc func(meta Meta) map[string]cli.CommandFactory

// Commands returns the mapping of CLI commands. The meta
// parameter lets you set meta options for all commands.
func Commands(metaPtr *Meta, agentUi cli.Ui, commands CommandFunc) map[string]cli.CommandFactory {
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

	all := map[string]cli.CommandFactory{}

	for k, v := range commands(meta) {
		all[k] = v
	}

	return all
}
