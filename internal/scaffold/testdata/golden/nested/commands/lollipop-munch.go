package commands

import (
	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const lollipopMunchFlags = command.FlagSetClient | command.FlagSetForce | command.FlagSetDryRun

type LollipopMunchCommand struct {
	command.Meta

	count int
}

func (c *LollipopMunchCommand) Name() string {
	return "lollipop munch"
}

func (c *LollipopMunchCommand) Synopsis() string {
	return "Eat lollipops from the jar"
}

func (c *LollipopMunchCommand) Description() string {
	return "Eating a lollipop removes it from the jar, so you are asked to confirm first. Pass --force to skip the question, or --dry-run to see what would be eaten."
}

func (c *LollipopMunchCommand) Deprecated() string {
	return "use \"nested lollipop eat\" instead; it will be removed in a future release"
}

func (c *LollipopMunchCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *LollipopMunchCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Eat a cherry lollipop", Command: appName + " " + c.Name() + " cherry"},
		{Description: "Eat two grape lollipops without being asked", Command: appName + " " + c.Name() + " grape --count 2 --force"},
		{Description: "See what would be eaten", Command: appName + " " + c.Name() + " lemon --dry-run", Output: "Would eat 1 lemon lollipop(s)"},
	}
}

func (c *LollipopMunchCommand) Arguments() []command.Argument {
	return []command.Argument{
		{
			Name:        "flavor",
			Description: "the flavor of lollipop to eat",
			Type:        command.ArgumentString,
			Prompt:      "Which flavor would you like?",
		},
	}
}

func (c *LollipopMunchCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictSet("cherry", "grape", "lemon", "lime")
}

func (c *LollipopMunchCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), lollipopMunchFlags)
	f.IntVar(&c.count, "count", 1, "number of lollipops to eat")
	c.EnvFlag(f, "count")
	return f
}

func (c *LollipopMunchCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(lollipopMunchFlags),
		complete.Flags{
			"--count": complete.PredictAnything,
		},
	)
}

func (c *LollipopMunchCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		// TODO: implement lollipop munch
		return nil
	})
}
