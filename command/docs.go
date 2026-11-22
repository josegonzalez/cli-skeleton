package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

// Doc formats supported by GenerateDocs.
const (
	DocsMarkdown = "markdown"
	DocsMan      = "man"
)

// documentedCommand is a command and the name it is registered under.
type documentedCommand struct {
	name string
	cmd  Command
}

// GenerateDocs writes reference docs for the app and every visible command
// to dir, built from the same synopsis, description, arguments, flags and
// examples as the help output.
//
// The markdown format writes `index.md` and one `<command-path>.md` per
// command, such as `lollipop-eat.md`, matching the links in help output.
// The man format writes `<app>.1` and one `<app>-<command-path>.1` per
// command, so `man <app>-<command-path>` works once installed.
func (m *Meta) GenerateDocs(dir string, format string, commands map[string]cli.CommandFactory) ([]string, error) {
	if format != DocsMarkdown && format != DocsMan {
		return nil, NewUsageError("unknown docs format %q; use %s or %s", format, DocsMarkdown, DocsMan)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	documented := m.documentedCommands(commands)
	var written []string
	write := func(name string, contents string) error {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			return err
		}
		written = append(written, path)
		return nil
	}

	if format == DocsMarkdown {
		if err := write("index.md", m.markdownIndex(documented)); err != nil {
			return written, err
		}
		for _, d := range documented {
			if err := write(commandPath(d.name)+".md", m.markdownCommand(d)); err != nil {
				return written, err
			}
		}
		return written, nil
	}

	if err := write(m.appName+".1", m.manIndex(documented)); err != nil {
		return written, err
	}
	for _, d := range documented {
		if err := write(m.appName+"-"+commandPath(d.name)+".1", m.manCommand(d)); err != nil {
			return written, err
		}
	}
	return written, nil
}

// documentedCommands returns the visible commands, sorted by name.
func (m *Meta) documentedCommands(commands map[string]cli.CommandFactory) []documentedCommand {
	names := make([]string, 0, len(commands))
	for name := range commands {
		if !m.isHidden(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var documented []documentedCommand
	for _, name := range names {
		factory := commands[name]
		if factory == nil {
			continue
		}
		raw, err := factory()
		if err != nil {
			continue
		}
		if cmd, ok := raw.(Command); ok {
			documented = append(documented, documentedCommand{name: name, cmd: cmd})
		}
	}
	return documented
}

// usageLine returns the usage of a command without the app name.
func usageLine(c Command) string {
	usage := c.Name()
	if c.FlagSet().HasFlags() {
		usage += " [flags]"
	}
	if args := ArgumentAsString(c.Arguments()); args != "" {
		usage += " " + args
	}
	return usage
}

// description returns the long description of a command, if any.
func description(c Command) string {
	if d, ok := c.(Describer); ok {
		return strings.TrimSpace(d.Description())
	}
	return ""
}

func (m *Meta) markdownIndex(documented []documentedCommand) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", m.appName)
	if m.description != "" {
		fmt.Fprintf(&b, "%s\n\n", m.description)
	}
	fmt.Fprintf(&b, "## Usage\n\n```\n%s <command> [flags]\n```\n\n", m.appName)
	if len(m.examples) > 0 {
		b.WriteString("## Examples\n\n")
		writeMarkdownExamples(&b, m.examples)
	}
	b.WriteString("## Commands\n\n")
	for _, d := range documented {
		fmt.Fprintf(&b, "- [`%s %s`](%s.md): %s\n", m.appName, d.name, commandPath(d.name), d.cmd.Synopsis())
	}
	return b.String()
}

func (m *Meta) markdownCommand(d documentedCommand) string {
	c := d.cmd
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", m.appName, d.name)
	fmt.Fprintf(&b, "%s\n\n", c.Synopsis())
	if text := description(c); text != "" {
		fmt.Fprintf(&b, "%s\n\n", text)
	}

	fmt.Fprintf(&b, "## Usage\n\n```\n%s %s\n```\n\n", m.appName, usageLine(c))

	if examples := c.Examples(); len(examples) > 0 {
		b.WriteString("## Examples\n\n")
		writeMarkdownExamples(&b, examples)
	}

	if arguments := c.Arguments(); len(arguments) > 0 {
		b.WriteString("## Arguments\n\n")
		for _, argument := range arguments {
			fmt.Fprintf(&b, "- `%s`: %s\n", ArgumentAsString([]Argument{argument}), argument.Description)
		}
		b.WriteString("\n")
	}

	if options := flagUsages(c.FlagSet(), 0); options != "" {
		fmt.Fprintf(&b, "## Options\n\n```\n%s\n```\n\n", options)
	}

	if aliases := m.aliasesFor(d.name); len(aliases) > 0 {
		fmt.Fprintf(&b, "## Aliases\n\n%s\n\n", "`"+strings.Join(aliases, "`, `")+"`")
	}

	fmt.Fprintf(&b, "See [all commands](index.md).\n")
	return b.String()
}

func writeMarkdownExamples(b *strings.Builder, examples []Example) {
	for _, example := range examples {
		if example.Description != "" {
			fmt.Fprintf(b, "%s:\n\n", example.Description)
		}
		fmt.Fprintf(b, "```shell\n$ %s\n", example.Command)
		if output := strings.TrimRight(example.Output, "\n"); output != "" {
			fmt.Fprintf(b, "%s\n", output)
		}
		b.WriteString("```\n\n")
	}
}

// roff escapes text for man pages.
func roff(text string) string {
	text = strings.ReplaceAll(text, `\`, `\e`)
	text = strings.ReplaceAll(text, "-", `\-`)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "'") {
			lines[i] = `\&` + line
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Meta) manHeader(b *strings.Builder, name string) {
	version := valueOr(m.version, "dev")
	fmt.Fprintf(b, ".TH \"%s\" \"1\" \"%s\" \"%s %s\" \"%s Manual\"\n", strings.ToUpper(roff(name)), m.docsDate(), roff(m.appName), roff(version), roff(m.appName))
}

// docsDate returns the date shown in man pages: SOURCE_DATE_EPOCH for
// reproducible builds, then the build date, then today.
func (m *Meta) docsDate() string {
	if epoch, err := strconv.ParseInt(os.Getenv("SOURCE_DATE_EPOCH"), 10, 64); err == nil {
		return time.Unix(epoch, 0).UTC().Format("2006-01-02")
	}
	if date := m.BuildInfo().BuildDate; len(date) >= 10 {
		return date[:10]
	}
	return timeNow().Format("2006-01-02")
}

func (m *Meta) manIndex(documented []documentedCommand) string {
	var b strings.Builder
	m.manHeader(&b, m.appName)
	fmt.Fprintf(&b, ".SH NAME\n%s", roff(m.appName))
	if m.description != "" {
		fmt.Fprintf(&b, " \\- %s", roff(m.description))
	}
	fmt.Fprintf(&b, "\n.SH SYNOPSIS\n.B %s\n<command> [flags]\n", roff(m.appName))
	if m.description != "" {
		fmt.Fprintf(&b, ".SH DESCRIPTION\n%s\n", roff(m.description))
	}
	b.WriteString(".SH COMMANDS\n")
	for _, d := range documented {
		fmt.Fprintf(&b, ".TP\n\\fB%s\\fR\n%s\n", roff(d.name), roff(d.cmd.Synopsis()))
	}
	if len(m.examples) > 0 {
		b.WriteString(".SH EXAMPLES\n")
		writeManExamples(&b, m.examples)
	}
	b.WriteString(".SH SEE ALSO\n")
	refs := make([]string, 0, len(documented))
	for _, d := range documented {
		refs = append(refs, fmt.Sprintf("\\fB%s\\fR(1)", roff(m.appName+"-"+commandPath(d.name))))
	}
	b.WriteString(strings.Join(refs, ", ") + "\n")
	return b.String()
}

func (m *Meta) manCommand(d documentedCommand) string {
	c := d.cmd
	var b strings.Builder
	m.manHeader(&b, m.appName+"-"+commandPath(d.name))
	fmt.Fprintf(&b, ".SH NAME\n%s \\- %s\n", roff(m.appName+"-"+commandPath(d.name)), roff(c.Synopsis()))
	fmt.Fprintf(&b, ".SH SYNOPSIS\n.B %s %s\n%s\n", roff(m.appName), roff(d.name), roff(strings.TrimPrefix(usageLine(c), c.Name())))

	fmt.Fprintf(&b, ".SH DESCRIPTION\n%s\n", roff(c.Synopsis()))
	if text := description(c); text != "" {
		fmt.Fprintf(&b, ".PP\n%s\n", roff(text))
	}

	if arguments := c.Arguments(); len(arguments) > 0 {
		b.WriteString(".SH ARGUMENTS\n")
		for _, argument := range arguments {
			fmt.Fprintf(&b, ".TP\n\\fI%s\\fR\n%s\n", roff(ArgumentAsString([]Argument{argument})), roff(argument.Description))
		}
	}

	flags := c.FlagSet()
	if flags.HasFlags() {
		b.WriteString(".SH OPTIONS\n")
		flags.VisitAll(func(f *flag.Flag) {
			if f.Hidden || f.Deprecated != "" {
				return
			}
			name := "\\fB\\-\\-" + roff(f.Name) + "\\fR"
			if f.Shorthand != "" {
				name = "\\fB\\-" + roff(f.Shorthand) + "\\fR, " + name
			}
			varname, usage := flag.UnquoteUsage(f)
			if varname != "" && f.Value.Type() != "bool" {
				name += " \\fI" + roff(varname) + "\\fR"
			}
			if def := flagDefault(f); def != "" {
				usage += " (default " + def + ")"
			}
			fmt.Fprintf(&b, ".TP\n%s\n%s\n", name, roff(usage))
		})
	}

	if examples := c.Examples(); len(examples) > 0 {
		b.WriteString(".SH EXAMPLES\n")
		writeManExamples(&b, examples)
	}

	fmt.Fprintf(&b, ".SH SEE ALSO\n\\fB%s\\fR(1)\n", roff(m.appName))
	return b.String()
}

func writeManExamples(b *strings.Builder, examples []Example) {
	for i, example := range examples {
		if example.Description != "" {
			if i > 0 {
				b.WriteString(".PP\n")
			}
			fmt.Fprintf(b, "%s\n", roff(example.Description))
		}
		fmt.Fprintf(b, ".RS 4\n.nf\n$ %s\n", roff(example.Command))
		if output := strings.TrimRight(example.Output, "\n"); output != "" {
			fmt.Fprintf(b, "%s\n", roff(output))
		}
		b.WriteString(".fi\n.RE\n")
	}
}

// manDir returns the per-user man page directory, which man searches by
// default on most systems.
func manDir() (string, error) {
	data, err := xdgBase("XDG_DATA_HOME", filepath.Join(".local", "share"))
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "man", "man1"), nil
}

// docsCommand generates reference docs. It is hidden from the command list.
type docsCommand struct {
	Meta

	commands map[string]cli.CommandFactory
	format   string
	output   string
	install  bool
}

func (c *docsCommand) Name() string          { return "docs" }
func (c *docsCommand) Help() string          { return CommandHelp(c) }
func (c *docsCommand) Arguments() []Argument { return []Argument{} }

func (c *docsCommand) Synopsis() string {
	return "Generate markdown or man page reference docs"
}

func (c *docsCommand) Description() string {
	return "Writes one page per command, built from the same text as the help output. With --install, man pages are written to the per-user man directory after asking for confirmation, so 'man " + c.AppName() + "-<command>' works."
}

func (c *docsCommand) Examples() []Example {
	app := c.AppName()
	return []Example{
		{Description: "Write markdown docs to ./docs", Command: app + " docs"},
		{Description: "Write man pages to ./man", Command: app + " docs --format man --output man"},
		{Description: "Install man pages for the current user", Command: app + " docs --format man --install"},
	}
}

func (c *docsCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), FlagSetClient|FlagSetForce)
	f.StringVar(&c.format, "format", DocsMarkdown, "the docs format: markdown or man")
	f.StringVar(&c.output, "output", "docs", "the directory to write the docs to")
	f.BoolVar(&c.install, "install", false, "install man pages for the current user")
	return f
}

func (c *docsCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *docsCommand) AutocompleteFlags() complete.Flags {
	return MergeAutocompleteFlags(c.Meta.AutocompleteFlags(FlagSetClient|FlagSetForce), complete.Flags{
		"--format":  complete.PredictSet(DocsMarkdown, DocsMan),
		"--output":  complete.PredictDirs("*"),
		"--install": complete.PredictNothing,
	})
}

func (c *docsCommand) Run(args []string) int {
	return Execute(c, args, func(map[string]Argument) error {
		dir := c.output
		if c.install {
			if c.format != DocsMan {
				return NewUsageError("--install requires --format man")
			}
			installDir, err := manDir()
			if err != nil {
				return err
			}
			dir = installDir

			ok, err := c.Confirm(fmt.Sprintf("Install man pages to %s?", dir))
			if err != nil {
				return err
			}
			if !ok {
				c.Ui.Info("Nothing was changed.")
				return nil
			}
		}

		written, err := c.GenerateDocs(dir, c.format, c.commands)
		if err != nil {
			return err
		}

		c.Success(fmt.Sprintf("Wrote %d pages to %s", len(written), dir))
		if c.install && !manpathIncludes(filepath.Dir(dir)) {
			c.Ui.Info(fmt.Sprintf("If 'man %s' does not find the pages, add %s to MANPATH.", c.AppName(), filepath.Dir(dir)))
		}
		return nil
	})
}

// manpathIncludes reports whether man searches dir, according to manpath.
func manpathIncludes(dir string) bool {
	out, err := exec.Command("manpath").Output()
	if err != nil {
		return false
	}
	for _, entry := range filepath.SplitList(strings.TrimSpace(string(out))) {
		if filepath.Clean(entry) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}
