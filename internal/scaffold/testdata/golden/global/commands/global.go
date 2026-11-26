package commands

import (
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type GlobalCommand struct {
	command.Meta
	GlobalFlagCommand
}

func (c *GlobalCommand) Name() string {
	return "global"
}

func (c *GlobalCommand) Synopsis() string {
	return "Global command that prints the values of the global flags"
}

func (c *GlobalCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *GlobalCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Prints the values of the global flags", Command: appName + " " + c.Name()},
	}
}

func (c *GlobalCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *GlobalCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *GlobalCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	c.GlobalFlags(f)
	return f
}

func (c *GlobalCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		c.AutocompleteGlobalFlags(),
		complete.Flags{},
	)
}

func (c *GlobalCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement global
		return nil
	})
}
