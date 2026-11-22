package command

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
)

// Example is a single usage example shown in help output.
type Example struct {
	// Description explains what the example does.
	Description string

	// Command is the full command line, starting with the app name.
	Command string

	// Output is optional sample output of the command.
	Output string
}

// Describer is implemented by commands with a long description. The
// description is shown in full help and in generated docs, below the
// one-line synopsis.
type Describer interface {
	Description() string
}

// helpOptions controls how help is rendered.
type helpOptions struct {
	// color makes headings bold.
	color bool

	// width wraps text to this many columns. Zero disables wrapping.
	width int

	// concise renders the short help shown on usage errors.
	concise bool
}

// CommandHelp returns the full help text for a command. Headings are bold
// when stdout is a color-capable terminal, and text wraps to the terminal
// width or COLUMNS.
func CommandHelp(c Command) string {
	m := c.meta()
	return renderCommandHelp(c, helpOptions{color: m.stdoutColor(), width: m.terminalWidth()})
}

// conciseCommandHelp returns the short help shown on stderr when a command
// is missing required arguments.
func conciseCommandHelp(c Command) string {
	m := c.meta()
	return renderCommandHelp(c, helpOptions{color: m.stderrColor(), width: m.terminalWidth(), concise: true})
}

func renderCommandHelp(c Command, o helpOptions) string {
	m := c.meta()
	h := &helpBuilder{color: o.color}

	usage := m.appName + " " + c.Name()
	flags := c.FlagSet()
	if flags.HasFlags() {
		usage += " [flags]"
	}
	if args := ArgumentAsString(c.Arguments()); args != "" {
		usage += " " + args
	}
	h.section("USAGE", indent(usage, 2))

	about := wrapText(c.Synopsis(), o.width)
	if d, ok := c.(Describer); ok && !o.concise {
		if description := strings.TrimSpace(d.Description()); description != "" {
			about += "\n\n" + wrapText(description, o.width)
		}
	}
	h.paragraph(about)

	examples := c.Examples()
	if o.concise && len(examples) > 1 {
		examples = examples[:1]
	}
	if len(examples) > 0 {
		h.section("EXAMPLES", ExampleString(examples))
	}

	if o.concise {
		h.paragraph(fmt.Sprintf("Run '%s %s --help' for more information.", m.appName, c.Name()))
		return h.String()
	}

	if arguments := argumentsString(c.Arguments(), o.width); arguments != "" {
		h.section("ARGUMENTS", arguments)
	}

	if options := flagUsages(flags, o.width); options != "" {
		h.section("OPTIONS", options)
	}

	if aliases := m.aliasesFor(c.Name()); len(aliases) > 0 {
		h.section("ALIASES", indent(strings.Join(aliases, ", "), 2))
	}

	h.learnMore(m, commandPath(c.Name()))
	return h.String()
}

// flagUsages renders the options section of help output, sorted by flag
// name, aligned, and wrapped to width columns. Hidden and deprecated flags
// are omitted.
func flagUsages(flags *flag.FlagSet, width int) string {
	type row struct {
		left  string
		usage string
	}

	var rows []row
	longest := 0
	flags.VisitAll(func(f *flag.Flag) {
		if f.Hidden || f.Deprecated != "" {
			return
		}

		left := "      --" + f.Name
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			left = "  -" + f.Shorthand + ", --" + f.Name
		}

		varname, usage := flag.UnquoteUsage(f)
		isBool := f.Value.Type() == "bool"
		if varname != "" && !isBool {
			left += " " + varname
		}
		if f.NoOptDefVal != "" && !isBool {
			left += "[=" + f.NoOptDefVal + "]"
		}

		if def := flagDefault(f); def != "" {
			usage += " (default " + def + ")"
		}

		rows = append(rows, row{left: left, usage: usage})
		longest = max(longest, len(left))
	})

	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		padding := strings.Repeat(" ", longest-len(r.left)+3)
		lines = append(lines, r.left+padding+wrap(longest+3, width, r.usage))
	}
	return strings.Join(lines, "\n")
}

// flagDefault returns the default value shown in help, or an empty string
// when the default is the zero value.
func flagDefault(f *flag.Flag) string {
	switch f.DefValue {
	case "", "0", "false", "[]", "<nil>", "0s", "map[]":
		return ""
	}
	if f.Value.Type() == "string" {
		return strconv.Quote(f.DefValue)
	}
	return f.DefValue
}

// ExampleString renders examples in the order given.
func ExampleString(examples []Example) string {
	blocks := make([]string, 0, len(examples))
	for _, example := range examples {
		lines := []string{}
		if example.Description != "" {
			lines = append(lines, "  "+example.Description)
		}
		lines = append(lines, "    $ "+example.Command)
		if output := strings.TrimRight(example.Output, "\n"); output != "" {
			lines = append(lines, indent(output, 4))
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

// commandPath turns a command name such as `lollipop eat` into the form
// used in URLs and file names, `lollipop-eat`.
func commandPath(name string) string {
	return strings.Join(strings.Fields(name), "-")
}

// helpBuilder assembles help text out of sections separated by blank lines.
type helpBuilder struct {
	color bool
	parts []string
}

func (h *helpBuilder) heading(title string) string {
	if h.color {
		return "\x1b[1m" + title + "\x1b[0m"
	}
	return title
}

func (h *helpBuilder) section(title string, body string) {
	h.parts = append(h.parts, h.heading(title)+"\n"+body)
}

func (h *helpBuilder) paragraph(text string) {
	if text = strings.TrimSpace(text); text != "" {
		h.parts = append(h.parts, text)
	}
}

// learnMore adds the docs and support links, when configured.
func (h *helpBuilder) learnMore(m *Meta, path string) {
	var lines []string
	if m.docsURL != "" {
		docs := strings.TrimRight(m.docsURL, "/")
		if path != "" {
			docs += "/" + path
		}
		lines = append(lines, "  Read the docs at "+docs)
	}
	if m.supportURL != "" {
		lines = append(lines, "  Get help or report issues at "+m.supportURL)
	}
	if len(lines) > 0 {
		h.section("LEARN MORE", strings.Join(lines, "\n"))
	}
}

func (h *helpBuilder) String() string {
	return strings.Join(h.parts, "\n\n") + "\n"
}

// indent prefixes every line of text with n spaces.
func indent(text string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// wrapText wraps each paragraph of text to width columns. Lines that start
// with whitespace are treated as preformatted. A width of zero disables
// wrapping.
func wrapText(text string, width int) string {
	text = strings.TrimSpace(text)
	if width <= 0 {
		return text
	}

	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || len(line) <= width {
			out = append(out, line)
			continue
		}

		current := ""
		for _, word := range strings.Fields(line) {
			if current == "" {
				current = word
			} else if len(current)+1+len(word) > width {
				out = append(out, current)
				current = word
			} else {
				current += " " + word
			}
		}
		if current != "" {
			out = append(out, current)
		}
	}
	return strings.Join(out, "\n")
}

// terminalWidth returns the width help text wraps to: COLUMNS when set,
// otherwise the width of stdout when it is a terminal, otherwise zero.
func (m *Meta) terminalWidth() int {
	return terminalWidth(m.stdoutFile)
}

func terminalWidth(f *os.File) int {
	if columns, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && columns > 0 {
		return columns
	}
	if f != nil && IsTerminal(f) {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return 0
}

// terminalHeight returns the height of the terminal: LINES when set,
// otherwise the height of f when it is a terminal, otherwise zero.
func terminalHeight(f *os.File) int {
	if lines, err := strconv.Atoi(os.Getenv("LINES")); err == nil && lines > 0 {
		return lines
	}
	if f != nil && IsTerminal(f) {
		if _, height, err := term.GetSize(int(f.Fd())); err == nil && height > 0 {
			return height
		}
	}
	return 0
}

// stdoutColor reports whether the UI colors stdout.
func (m *Meta) stdoutColor() bool {
	if m.Ui == nil {
		return false
	}
	stdout, _ := uiColors(m.Ui)
	return stdout
}

// stderrColor reports whether the UI colors stderr.
func (m *Meta) stderrColor() bool {
	if m.Ui == nil {
		return false
	}
	_, stderr := uiColors(m.Ui)
	return stderr
}

// HelpTemplate implements cli.CommandHelpTemplate, so the subcommand list
// mitchellh/cli appends to the help of parent commands matches the rest of
// the help output.
func (m *Meta) HelpTemplate() string {
	h := &helpBuilder{color: m.stdoutColor()}
	return `{{.Help | trim}}{{if gt (len .Subcommands) 0}}

` + h.heading("SUBCOMMANDS") + `
{{- range $value := .Subcommands }}
  {{ $value.NameAligned }}   {{ $value.Synopsis }}{{ end }}{{ end }}`
}

// topLevelHelp renders the help shown by `app --help`, or the concise help
// shown on stderr when the app is run without arguments.
func (m *Meta) topLevelHelp(commands map[string]cli.CommandFactory, concise bool) string {
	color := m.stdoutColor()
	if concise {
		color = m.stderrColor()
	}
	width := m.terminalWidth()
	h := &helpBuilder{color: color}

	if m.description != "" {
		h.paragraph(wrapText(m.description, width))
	}

	h.section("USAGE", indent(m.appName+" <command> [flags]", 2))

	examples := m.examples
	if concise && len(examples) > 2 {
		examples = examples[:2]
	}
	if len(examples) > 0 {
		h.section("EXAMPLES", ExampleString(examples))
	}

	if list := commandList(commands); list != "" {
		h.section("COMMANDS", list)
	}

	if concise {
		h.paragraph(fmt.Sprintf("Run '%s --help' for more information.", m.appName))
		return h.String()
	}

	h.section("OPTIONS", strings.Join([]string{
		"  -h, --help      show help for the app or a command",
		"  -v, --version   show the version",
	}, "\n"))

	h.paragraph(fmt.Sprintf("Run '%s help <command>' or '%s <command> --help' for more information on a command.", m.appName, m.appName))
	h.learnMore(m, "")
	return h.String()
}

// commandList renders commands and their synopsis, aligned and sorted.
func commandList(commands map[string]cli.CommandFactory) string {
	names := make([]string, 0, len(commands))
	longest := 0
	for name := range commands {
		names = append(names, name)
		longest = max(longest, len(name))
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		synopsis := ""
		if factory := commands[name]; factory != nil {
			if c, err := factory(); err == nil && c != nil {
				synopsis = c.Synopsis()
			}
		}
		lines = append(lines, fmt.Sprintf("  %-*s   %s", longest, name, synopsis))
	}
	return strings.Join(lines, "\n")
}

// visibleCommands returns the commands shown in the command list.
func (m *Meta) visibleCommands(commands map[string]cli.CommandFactory) map[string]cli.CommandFactory {
	visible := map[string]cli.CommandFactory{}
	for name, factory := range commands {
		if strings.Contains(name, " ") || m.isHidden(name) {
			continue
		}
		visible[name] = factory
	}

	// Parents of nested commands are listed even without their own factory.
	for name := range commands {
		if parent, _, ok := strings.Cut(name, " "); ok && !m.isHidden(parent) {
			if _, exists := visible[parent]; !exists {
				visible[parent] = nil
			}
		}
	}
	return visible
}

// isHidden reports whether a command is hidden from the command list.
func (m *Meta) isHidden(name string) bool {
	for _, hidden := range m.hiddenCommands {
		if hidden == name {
			return true
		}
	}
	return false
}

// aliasesFor returns the aliases of a command.
func (m *Meta) aliasesFor(name string) []string {
	var aliases []string
	for alias, target := range m.aliases {
		if target == name {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

// rewriteHelpArgs turns `help` into `--help` and `help <command>` into
// `<command> --help`, unless the app defines its own help command.
func rewriteHelpArgs(args []string, commands map[string]cli.CommandFactory) []string {
	if len(args) == 0 || args[0] != "help" {
		return args
	}
	if _, ok := commands["help"]; ok {
		return args
	}

	var words, flags []string
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else {
			words = append(words, arg)
		}
	}

	result := append(words, flags...)
	return append(result, "--help")
}

// unknownCommand checks whether args name a command. It returns the
// unrecognized word and the commands it was checked against, or an empty
// string when the command exists or args start with a flag.
func unknownCommand(args []string, commands map[string]cli.CommandFactory) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil
	}

	prefix := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return "", nil
		}

		candidates := childCommands(prefix, commands)
		if len(candidates) == 0 {
			return "", nil
		}

		name := strings.TrimSpace(prefix + " " + arg)
		_, exists := commands[name]
		if !exists && !hasChildren(name, commands) {
			if prefix != "" {
				if _, parentExists := commands[prefix]; parentExists {
					// A parent command with its own factory may take
					// positional arguments.
					return "", nil
				}
			}
			return arg, candidates
		}

		if !hasChildren(name, commands) {
			return "", nil
		}
		prefix = name
	}
	return "", nil
}

// childCommands returns the next word of every command under prefix.
func childCommands(prefix string, commands map[string]cli.CommandFactory) []string {
	seen := map[string]bool{}
	var children []string
	for name := range commands {
		rest := name
		if prefix != "" {
			if !strings.HasPrefix(name, prefix+" ") {
				continue
			}
			rest = name[len(prefix)+1:]
		}
		word, _, _ := strings.Cut(rest, " ")
		if word != "" && !seen[word] {
			seen[word] = true
			children = append(children, word)
		}
	}
	sort.Strings(children)
	return children
}

// hasChildren reports whether name has nested commands.
func hasChildren(name string, commands map[string]cli.CommandFactory) bool {
	for key := range commands {
		if strings.HasPrefix(key, name+" ") {
			return true
		}
	}
	return false
}

// examplesCommand prints the examples of every command, as a cheat sheet.
type examplesCommand struct {
	Meta

	commands map[string]cli.CommandFactory
}

func (c *examplesCommand) Name() string             { return "examples" }
func (c *examplesCommand) Synopsis() string         { return "Show examples for every command" }
func (c *examplesCommand) Help() string             { return CommandHelp(c) }
func (c *examplesCommand) Arguments() []Argument    { return []Argument{} }
func (c *examplesCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.Name(), FlagSetClient)
}

func (c *examplesCommand) Examples() []Example {
	return []Example{
		{Description: "Show examples for every command", Command: c.AppName() + " examples"},
	}
}

func (c *examplesCommand) Run(args []string) int {
	return Execute(c, args, func(map[string]Argument) error {
		h := &helpBuilder{color: c.stdoutColor()}

		names := make([]string, 0, len(c.commands))
		for name := range c.commands {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			if name == c.Name() || c.isHidden(name) {
				continue
			}
			raw, err := c.commands[name]()
			if err != nil {
				continue
			}
			cmd, ok := raw.(Command)
			if !ok {
				continue
			}
			if examples := cmd.Examples(); len(examples) > 0 {
				h.section(strings.ToUpper(name), ExampleString(examples))
			}
		}

		c.Ui.Output(strings.TrimRight(h.String(), "\n"))
		return nil
	})
}

// parentCommand groups nested commands, such as `lollipop` for
// `lollipop eat` and `lollipop list`.
type parentCommand struct {
	Meta

	name     string
	commands map[string]cli.CommandFactory
}

func (c *parentCommand) Name() string          { return c.name }
func (c *parentCommand) Arguments() []Argument { return []Argument{} }
func (c *parentCommand) Examples() []Example   { return []Example{} }

func (c *parentCommand) Synopsis() string {
	return "Commands for " + c.name
}

func (c *parentCommand) FlagSet() *flag.FlagSet {
	return c.Meta.FlagSet(c.name, FlagSetNone)
}

// Help returns the parent help. mitchellh/cli appends the subcommand list.
func (c *parentCommand) Help() string {
	h := &helpBuilder{color: c.stdoutColor()}
	h.section("USAGE", indent(c.appName+" "+c.name+" <command> [flags]", 2))
	h.paragraph(c.Synopsis())
	h.learnMore(&c.Meta, commandPath(c.name))
	return h.String()
}

// Run prints the parent help and its subcommands to stderr, since a
// subcommand is required.
func (c *parentCommand) Run(args []string) int {
	h := &helpBuilder{color: c.stderrColor()}
	h.section("USAGE", indent(c.appName+" "+c.name+" <command> [flags]", 2))
	h.paragraph(c.Synopsis())

	children := map[string]cli.CommandFactory{}
	for name, factory := range c.commands {
		if rest, ok := strings.CutPrefix(name, c.name+" "); ok && !strings.Contains(rest, " ") && !c.isHidden(name) {
			children[rest] = factory
		}
	}
	if list := commandList(children); list != "" {
		h.section("SUBCOMMANDS", list)
	}
	h.paragraph(fmt.Sprintf("Run '%s %s <command> --help' for more information on a command.", c.appName, c.name))

	c.writeErr(h.String())
	return ExitCodeUsage
}
