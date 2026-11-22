package commands

import (
	"strconv"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const listFlags = command.FlagSetClient | command.FlagSetJSON | command.FlagSetPlain | command.FlagSetAll

type ListCommand struct {
	command.Meta
}

func (c *ListCommand) Name() string {
	return "lollipop list"
}

func (c *ListCommand) Synopsis() string {
	return "List the lollipops in the jar"
}

func (c *ListCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *ListCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{
			Description: "List the lollipops left in the jar",
			Command:     appName + " " + c.Name(),
			Output:      "FLAVOR   COLOR\ncherry   red\ngrape    purple\nlemon    yellow",
		},
		{Description: "Include the lollipops that were eaten", Command: appName + " " + c.Name() + " --all"},
		{Description: "Print the flavors for scripts", Command: appName + " " + c.Name() + " --plain | cut -f1"},
		{Description: "Print the lollipops as JSON", Command: appName + " " + c.Name() + " --json"},
	}
}

func (c *ListCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *ListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *ListCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), listFlags)
}

func (c *ListCommand) AutocompleteFlags() complete.Flags {
	return c.Meta.AutocompleteFlags(listFlags)
}

func (c *ListCommand) Run(args []string) int {
	return command.Execute(c, args, func(map[string]command.Argument) error {
		headers := []string{"flavor", "color"}
		if c.IsAll() {
			headers = append(headers, "eaten")
		}

		rows := [][]string{}
		for _, l := range lollipops {
			if l.Eaten && !c.IsAll() {
				continue
			}
			row := []string{l.Flavor, l.Color}
			if c.IsAll() {
				row = append(row, strconv.FormatBool(l.Eaten))
			}
			rows = append(rows, row)
		}

		return c.OutputTable(headers, rows)
	})
}
