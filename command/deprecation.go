package command

import (
	"fmt"
	"sort"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// Deprecated is implemented by commands that are being phased out. The
// message, such as `use "lollipop eat" instead`, is shown as a warning
// before the command runs, and the command is hidden from the command list
// while it keeps working.
type Deprecated interface {
	Deprecated() string
}

// DeprecateFlag marks a flag as deprecated. It is hidden from help, keeps
// working, and prints a warning to stderr only when it is used. Pass the
// name of the flag that replaces it, or an empty string when there is none.
//
// Call it from a command's FlagSet method, after defining the flag.
func DeprecateFlag(fs *flag.FlagSet, name string, replacement string) error {
	message := "it will be removed in a future release"
	if replacement != "" {
		message = fmt.Sprintf("use --%s instead; --%s will be removed in a future release", replacement, name)
	}
	return fs.MarkDeprecated(name, message)
}

// WithAliases registers explicit, stable aliases, mapping each alias to the
// command it runs, such as `{"rm": "remove"}`. Aliases are hidden from the
// command list and shown in the help of their command. Commands are never
// matched by prefix, so new commands can be added without breaking
// scripts.
func WithAliases(aliases map[string]string) Option {
	return func(o *mainOptions) {
		o.aliases = aliases
	}
}

// warnIfDeprecated prints the deprecation message of c, if any.
func (m *Meta) warnIfDeprecated(c Command) {
	d, ok := c.(Deprecated)
	if !ok {
		return
	}
	if message := d.Deprecated(); message != "" {
		m.Ui.Warn(fmt.Sprintf("Command %q is deprecated: %s", c.Name(), message))
	}
}

// addAliases registers the aliases of the app as hidden commands.
func addAliases(meta *Meta, commands map[string]cli.CommandFactory, aliases map[string]string) {
	if meta.registry == nil {
		meta.registry = &commandRegistry{}
	}
	meta.registry.aliases = map[string]string{}

	names := make([]string, 0, len(aliases))
	for alias := range aliases {
		names = append(names, alias)
	}
	sort.Strings(names)

	for _, alias := range names {
		target := aliases[alias]
		factory, ok := commands[target]
		if !ok {
			meta.Debugf("ignoring alias %q for unknown command %q", alias, target)
			continue
		}
		if _, exists := commands[alias]; exists {
			meta.Debugf("ignoring alias %q, which is already a command", alias)
			continue
		}

		commands[alias] = factory
		meta.registry.aliases[alias] = target
		meta.registry.hidden = append(meta.registry.hidden, alias)
	}
}

// hideDeprecatedCommands hides deprecated commands from the command list.
func hideDeprecatedCommands(meta *Meta, commands map[string]cli.CommandFactory) {
	if meta.registry == nil {
		meta.registry = &commandRegistry{}
	}

	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		factory := commands[name]
		if factory == nil {
			continue
		}
		c, err := factory()
		if err != nil {
			continue
		}
		if d, ok := c.(Deprecated); ok && d.Deprecated() != "" && !meta.isHidden(name) {
			meta.registry.hidden = append(meta.registry.hidden, name)
		}
	}
}

// visibleNames removes hidden commands from names.
func (m *Meta) visibleNames(names []string) []string {
	visible := make([]string, 0, len(names))
	for _, name := range names {
		if !m.isHidden(name) {
			visible = append(visible, name)
		}
	}
	return visible
}
