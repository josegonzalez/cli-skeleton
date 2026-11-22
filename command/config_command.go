package command

import (
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// configCommand shows the effective configuration and where each value
// comes from. It is added by WithConfig.
type configCommand struct {
	Meta

	commands map[string]cli.CommandFactory
}

// configEntry is one configured value.
type configEntry struct {
	Command string `json:"command,omitempty"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Source  string `json:"source"`
}

// configFileEntry is one config file location.
type configFileEntry struct {
	Path  string `json:"path"`
	Found bool   `json:"found"`
}

func (c *configCommand) Name() string          { return "config" }
func (c *configCommand) Arguments() []Argument { return []Argument{} }
func (c *configCommand) Help() string          { return CommandHelp(c) }

func (c *configCommand) Synopsis() string {
	return "Show the effective configuration and where it comes from"
}

func (c *configCommand) Description() string {
	return "Values are read from the system config, the user config, the project config and the environment, each overriding the last. Flags passed on the command line override them all."
}

func (c *configCommand) Examples() []Example {
	return []Example{
		{Description: "Show the configuration", Command: c.AppName() + " config"},
		{Description: "Show the configuration as JSON", Command: c.AppName() + " config --json"},
	}
}

func (c *configCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), FlagSetClient|FlagSetJSON|FlagSetPlain)
}

func (c *configCommand) Run(args []string) int {
	return Execute(c, args, func(map[string]Argument) error {
		files := c.configFiles()
		entries := c.configEntries()

		if c.IsJSON() {
			return c.OutputJSON(map[string]interface{}{
				"files":  files,
				"values": entries,
			})
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			key := e.Key
			if e.Command != "" {
				key = e.Command + "." + e.Key
			}
			rows = append(rows, []string{key, e.Value, e.Source})
		}

		if c.IsPlain() {
			return c.OutputTable([]string{"key", "value", "source"}, rows)
		}

		h := &helpBuilder{color: c.stdoutColor()}
		fileLines := make([]string, 0, len(files))
		for _, f := range files {
			status := ""
			if !f.Found {
				status = " (not found)"
			}
			fileLines = append(fileLines, "  "+f.Path+status)
		}
		h.section("FILES", strings.Join(fileLines, "\n"))
		c.writeOut(h.String())

		if len(rows) == 0 {
			c.Ui.Info("No values are configured.")
			return nil
		}
		return c.OutputTable([]string{"key", "value", "source"}, rows)
	})
}

// configFiles lists the config files that are read, lowest precedence
// first.
func (c *configCommand) configFiles() []configFileEntry {
	files := []configFileEntry{}
	if c.config == nil {
		return files
	}
	for _, f := range c.config.files {
		files = append(files, configFileEntry{Path: f.path, Found: f.found})
	}
	return files
}

// configEntries lists the configured values, from config files and from
// environment variables bound to flags.
func (c *configCommand) configEntries() []configEntry {
	entries := []configEntry{}
	seen := map[string]int{}

	add := func(e configEntry) {
		id := e.Command + "\x00" + e.Key
		if i, ok := seen[id]; ok {
			entries[i] = e
			return
		}
		seen[id] = len(entries)
		entries = append(entries, e)
	}

	if c.config != nil {
		for key, value := range c.config.global {
			add(configEntry{Key: key, Value: value.value, Source: value.source})
		}
		for command, section := range c.config.commands {
			for key, value := range section {
				add(configEntry{Command: command, Key: key, Value: value.value, Source: value.source})
			}
		}
	}

	for name, factory := range c.commands {
		if name == c.Name() || factory == nil || c.isHidden(name) {
			continue
		}
		raw, err := factory()
		if err != nil {
			continue
		}
		cmd, ok := raw.(Command)
		if !ok {
			continue
		}
		cmd.FlagSet()
		for flagName, env := range cmd.meta().envFlags {
			if value, set := os.LookupEnv(env); set {
				add(configEntry{Command: commandPath(name), Key: flagName, Value: value, Source: "env " + env})
			}
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Command != entries[j].Command {
			return entries[i].Command < entries[j].Command
		}
		return entries[i].Key < entries[j].Key
	})
	return entries
}

