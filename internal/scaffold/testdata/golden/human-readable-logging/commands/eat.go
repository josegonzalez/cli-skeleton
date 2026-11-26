package commands

import (
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type EatCommand struct {
	command.Meta

	count int
	color string
}

func (c *EatCommand) Name() string {
	return "eat"
}

func (c *EatCommand) Synopsis() string {
	return "Eats one or more lollipops"
}

func (c *EatCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *EatCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Eats one lollipop quickly", Command: appName + " " + c.Name() + " quickly"},
		{Description: "Eats one lollipop slowly", Command: appName + " " + c.Name() + " slowly"},
		{Description: "Eats two lollipops quickly", Command: appName + " " + c.Name() + " --count 2 quickly"},
		{Description: "Eats three red lollipops", Command: appName + " " + c.Name() + " --count 3 --color red"},
	}
}

func (c *EatCommand) Arguments() []command.Argument {
	return []command.Argument{
		{
			Name:        "speed",
			Description: "how quickly to eat the lollipop",
			Optional:    true,
			Type:        command.ArgumentString,
		},
	}
}

func (c *EatCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *EatCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), command.FlagSetClient)
	f.IntVar(&c.count, "count", 1, "number of lollipops to eat")
	f.StringVar(&c.color, "color", "normal", "the color of the lollipops being eaten")
	return f
}

func (c *EatCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(command.FlagSetClient),
		complete.Flags{
			"--count": complete.PredictAnything,
			"--color": complete.PredictSet("red", "orange", "yellow", "green", "blue", "purple"),
		},
	)
}

func (c *EatCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement eat
		return nil
	})
}
