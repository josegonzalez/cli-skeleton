package commands

import (
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const lollipopListFlags = command.FlagSetClient | command.FlagSetJSON | command.FlagSetPlain | command.FlagSetAll

type LollipopListCommand struct {
	command.Meta
}

func (c *LollipopListCommand) Name() string {
	return "lollipop list"
}

func (c *LollipopListCommand) Synopsis() string {
	return "List the lollipops in the jar"
}

func (c *LollipopListCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *LollipopListCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "List the lollipops left in the jar", Command: appName + " " + c.Name(), Output: "FLAVOR   COLOR\ncherry   red\ngrape    purple\nlemon    yellow"},
		{Description: "Include the lollipops that were eaten", Command: appName + " " + c.Name() + " --all"},
		{Description: "Print the flavors for scripts", Command: appName + " " + c.Name() + " --plain | cut -f1"},
		{Description: "Print the lollipops as JSON", Command: appName + " " + c.Name() + " --json"},
	}
}

func (c *LollipopListCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *LollipopListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *LollipopListCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), lollipopListFlags)
	return f
}

func (c *LollipopListCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(lollipopListFlags),
		complete.Flags{},
	)
}

func (c *LollipopListCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement lollipop list
		return nil
	})
}
