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
		c.Meta.AutocompleteFlags(FlagSetClient|FlagSetJSON),
		complete.Flags{},
	)
}

func (c *VersionCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *VersionCommand) Examples() []Example {
	return []Example{
		{Description: "Return the version of the binary", Command: fmt.Sprintf("%s %s", c.AppName(), c.Name())},
		{Description: "Return the version and build details as JSON", Command: fmt.Sprintf("%s %s --json", c.AppName(), c.Name())},
	}
}

func (c *VersionCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), FlagSetClient|FlagSetJSON)
}

func (c *VersionCommand) Name() string {
	return "version"
}

func (c *VersionCommand) Synopsis() string {
	return "Return the version of the binary"
}

func (c *VersionCommand) Run(args []string) int {
	return Execute(c, args, func(arguments map[string]Argument) error {
		info := c.BuildInfo()
		if c.IsJSON() {
			return c.OutputJSON(info)
		}

		c.Ui.Output(c.Version())
		if c.IsDebug() {
			c.Ui.Info(fmt.Sprintf("Go version: %s", info.GoVersion))
			c.Ui.Info(fmt.Sprintf("Platform: %s/%s", info.OS, info.Arch))
			if info.Commit != "" {
				c.Ui.Info(fmt.Sprintf("Commit: %s", info.Commit))
			}
			if info.BuildDate != "" {
				c.Ui.Info(fmt.Sprintf("Build date: %s", info.BuildDate))
			}
		}
		return nil
	})
}
