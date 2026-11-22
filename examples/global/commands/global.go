package commands

import (
	"fmt"
	"strconv"

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
		{Description: "Prints the values of the global flags", Command: fmt.Sprintf("%s %s", appName, c.Name())},
	}
}

func (c *GlobalCommand) Arguments() []command.Argument {
	args := []command.Argument{}
	return args
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
	return command.Execute(c, args, func(_ map[string]command.Argument) error {
		c.Ui.Output(fmt.Sprintf("Global bool value: %s", strconv.FormatBool(c.global)))
		c.Ui.Output(fmt.Sprintf("Global string value: %s", c.globalValue))

		return nil
	})
}
