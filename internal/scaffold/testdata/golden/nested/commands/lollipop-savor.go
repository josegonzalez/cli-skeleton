package commands

import (
	"time"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const lollipopSavorFlags = command.FlagSetClient | command.FlagSetTimeout

type LollipopSavorCommand struct {
	command.Meta

	duration time.Duration
	wait     bool
	drop     bool
}

func (c *LollipopSavorCommand) Name() string {
	return "lollipop savor"
}

func (c *LollipopSavorCommand) Synopsis() string {
	return "Slowly savor a lollipop"
}

func (c *LollipopSavorCommand) Description() string {
	return "Savoring takes a while, so progress is shown as it happens. Only one lollipop can be savored at a time; pass --wait to wait for another one to finish. Press Ctrl+C to stop."
}

func (c *LollipopSavorCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *LollipopSavorCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Savor a lollipop", Command: appName + " " + c.Name()},
		{Description: "Savor a lollipop for ten seconds", Command: appName + " " + c.Name() + " --duration 10s"},
		{Description: "Wait for another lollipop to be finished first", Command: appName + " " + c.Name() + " --wait"},
	}
}

func (c *LollipopSavorCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *LollipopSavorCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *LollipopSavorCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), lollipopSavorFlags)
	f.DurationVar(&c.duration, "duration", 3*time.Second, "how long to savor the lollipop")
	f.BoolVar(&c.wait, "wait", false, "wait for another lollipop to be finished instead of failing")
	f.BoolVar(&c.drop, "drop", false, "drop the lollipop halfway through, to show a failed step")
	return f
}

func (c *LollipopSavorCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(lollipopSavorFlags),
		complete.Flags{
			"--duration": complete.PredictAnything,
			"--wait":     complete.PredictNothing,
			"--drop":     complete.PredictNothing,
		},
	)
}

func (c *LollipopSavorCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement lollipop savor
		return nil
	})
}
