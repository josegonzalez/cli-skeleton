package command

import (
	"fmt"

	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type VersionCommand struct {
	Meta
}

func (c *VersionCommand) Help() string {
	return CommandHelp(c)
}

func (c *VersionCommand) Arguments() []Argument {
	args := []Argument{}
	return args
}

func (c *VersionCommand) AutocompleteFlags() complete.Flags {
	return MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(FlagSetClient),
		complete.Flags{},
	)
}

func (c *VersionCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *VersionCommand) Examples() map[string]string {
	return map[string]string{
		"Return the version of the binary": fmt.Sprintf("%s %s", c.AppName(), c.Name()),
	}
}

func (c *VersionCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), FlagSetClient)
}

func (c *VersionCommand) Name() string {
	return "version"
}

func (c *VersionCommand) Synopsis() string {
	return "Return the version of the binary"
}

func (c *VersionCommand) Run(args []string) int {
	return Execute(c, args, func(arguments map[string]Argument) error {
		c.Ui.Output(c.Version())
		return nil
	})
}
