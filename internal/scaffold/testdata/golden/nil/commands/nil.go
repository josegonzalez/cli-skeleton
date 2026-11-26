package commands

import (
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type NilCommand struct {
	command.Meta
}

func (c *NilCommand) Name() string {
	return "nil"
}

func (c *NilCommand) Synopsis() string {
	return "Nil command"
}

func (c *NilCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *NilCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Does nothing", Command: appName + " " + c.Name()},
	}
}

func (c *NilCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *NilCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *NilCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	return f
}

func (c *NilCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{},
	)
}

func (c *NilCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement nil
		return nil
	})
}
