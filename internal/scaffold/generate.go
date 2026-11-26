package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"quote": strconv.Quote,
}).ParseFS(templateFS, "templates/*.tmpl"))

// SkeletonModule is the module path of cli-skeleton.
const SkeletonModule = "github.com/josegonzalez/cli-skeleton"

// localVersion is the version required when cli-skeleton is replaced by a
// local checkout, matching the examples.
const localVersion = "v0.0.0-00010101000000-000000000000"

// Options changes how a project is generated.
type Options struct {
	// SkeletonPath, when set, replaces cli-skeleton with the checkout at
	// this path in go.mod.
	SkeletonPath string
}

// File is a generated file.
type File struct {
	// Path is relative to the project directory and uses forward slashes.
	Path     string
	Contents []byte
}

// Generate returns the files of the project described by spec. It does not
// write anything.
func Generate(spec *Spec, opts Options) ([]File, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	app := newAppData(spec, opts)
	var files []File
	add := func(name string, tmpl string, data interface{}, gofmt bool) error {
		var buf bytes.Buffer
		if err := templates.ExecuteTemplate(&buf, tmpl, data); err != nil {
			return fmt.Errorf("rendering %s: %w", name, err)
		}
		contents := buf.Bytes()
		if gofmt {
			formatted, err := format.Source(contents)
			if err != nil {
				return fmt.Errorf("formatting %s: %w\n%s", name, err, contents)
			}
			contents = formatted
		}
		files = append(files, File{Path: name, Contents: contents})
		return nil
	}

	if err := add("go.mod", "go.mod.tmpl", app, false); err != nil {
		return nil, err
	}
	if err := add(".gitignore", "gitignore.tmpl", app, false); err != nil {
		return nil, err
	}
	if err := add("main.go", "main.go.tmpl", app, true); err != nil {
		return nil, err
	}
	if err := add("main_test.go", "main_test.go.tmpl", app, true); err != nil {
		return nil, err
	}
	if app.Global {
		if err := add("commands/global_flags.go", "global_flags.go.tmpl", app, true); err != nil {
			return nil, err
		}
	}
	for _, c := range app.Commands {
		if err := add(path.Join("commands", c.File), "command.go.tmpl", c, true); err != nil {
			return nil, err
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

type appData struct {
	Name         string
	Module       string
	SkeletonPath string
	LocalVersion string
	Options      []string
	Commands     []commandData
	Global       bool
	GlobalFlags  []flagData
	GlobalTime   bool
}

type commandData struct {
	Name           string
	Type           string
	File           string
	Synopsis       string
	Description    string
	Deprecated     string
	FlagSetsConst  string
	FlagSetsExpr   string
	FlagSetsRef    string
	Arguments      []argumentData
	ArgumentValues string
	Flags          []flagData
	Examples       []string
	Global         bool
	UsesTime       bool
}

type argumentData struct {
	Name        string
	Description string
	Type        string
	Optional    bool
	Prompt      string
}

type flagData struct {
	Name       string
	Field      string
	GoType     string
	Func       string
	Default    string
	Usage      string
	Env        bool
	Completion string
}

func newAppData(spec *Spec, opts Options) appData {
	app := appData{
		Name:         spec.Name,
		Module:       spec.ModulePath(),
		SkeletonPath: opts.SkeletonPath,
		LocalVersion: localVersion,
		Global:       len(spec.GlobalFlags) > 0,
	}

	for _, f := range spec.GlobalFlags {
		fd := newFlagData(f)
		app.GlobalFlags = append(app.GlobalFlags, fd)
		if fd.GoType == "time.Duration" {
			app.GlobalTime = true
		}
	}

	app.Options = appOptions(spec)

	commands := make([]Command, len(spec.Commands))
	copy(commands, spec.Commands)
	sort.SliceStable(commands, func(i, j int) bool { return commands[i].Name < commands[j].Name })
	for _, c := range commands {
		app.Commands = append(app.Commands, newCommandData(c, app.Global))
	}
	return app
}

// appOptions returns the command.Main options of the app, as Go source.
func appOptions(spec *Spec) []string {
	var options []string
	if spec.Description != "" {
		options = append(options, fmt.Sprintf("command.WithDescription(%s)", strconv.Quote(spec.Description)))
	}
	if len(spec.Examples) > 0 {
		var b strings.Builder
		b.WriteString("command.WithExamples(\n")
		for _, example := range spec.Examples {
			b.WriteString("command.Example{")
			b.WriteString(exampleFields(example.Description, "AppName", example.Command, example.Output))
			b.WriteString("},\n")
		}
		b.WriteString(")")
		options = append(options, b.String())
	}
	switch spec.UI {
	case UIZerolog:
		options = append(options, "command.WithUi(func(ui cli.Ui) cli.Ui {\nreturn command.ZerologUiWithFields(ui, make(map[string]interface{}, 0))\n})")
	case UIHumanZerolog:
		options = append(options, "command.WithUi(func(ui cli.Ui) cli.Ui {\nreturn command.HumanZerologUiWithFields(ui, make(map[string]interface{}, 0))\n})")
	}
	if len(spec.GlobalFlags) > 0 {
		options = append(options, "command.WithGlobalFlags((&commands.GlobalFlagCommand{}).GlobalFlags)")
	}
	if len(spec.Aliases) > 0 {
		aliases := make([]string, 0, len(spec.Aliases))
		for alias := range spec.Aliases {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)

		var b strings.Builder
		b.WriteString("command.WithAliases(map[string]string{\n")
		for _, alias := range aliases {
			fmt.Fprintf(&b, "%s: %s,\n", strconv.Quote(alias), strconv.Quote(spec.Aliases[alias]))
		}
		b.WriteString("})")
		options = append(options, b.String())
	}
	if spec.Config {
		options = append(options, "command.WithConfig()")
	}
	if spec.DocsURL != "" {
		options = append(options, fmt.Sprintf("command.WithDocsURL(%s)", strconv.Quote(spec.DocsURL)))
	}
	if spec.SupportURL != "" {
		options = append(options, fmt.Sprintf("command.WithSupportURL(%s)", strconv.Quote(spec.SupportURL)))
	}
	if spec.IssueURL != "" {
		options = append(options, fmt.Sprintf("command.WithIssueURL(%s)", strconv.Quote(spec.IssueURL)))
	}
	return options
}

// exampleFields returns the fields of a command.Example literal. prefix is
// the expression the command starts with, and rest is appended to it.
func exampleFields(description string, prefix string, rest string, output string) string {
	commandExpr := prefix
	if rest = strings.TrimSpace(rest); rest != "" {
		commandExpr += " + " + strconv.Quote(" "+rest)
	}
	fields := fmt.Sprintf("Description: %s, Command: %s", strconv.Quote(description), commandExpr)
	if output != "" {
		fields += ", Output: " + strconv.Quote(output)
	}
	return fields
}

func newCommandData(c Command, global bool) commandData {
	data := commandData{
		Name:        c.Name,
		Type:        TypeName(c.Name),
		File:        FileName(c.Name),
		Synopsis:    c.Synopsis,
		Description: c.Description,
		Deprecated:  c.Deprecated,
		Global:      global,
	}

	flagSets := c.FlagSets
	if len(flagSets) == 0 {
		flagSets = []string{"client"}
	}
	bits := make([]string, 0, len(flagSets))
	seen := map[string]bool{}
	for _, name := range flagSets {
		if !seen[name] {
			seen[name] = true
			bits = append(bits, FlagSets[name])
		}
	}
	if len(bits) == 1 {
		data.FlagSetsRef = bits[0]
	} else {
		data.FlagSetsConst = flagSetsConst(c.Name)
		data.FlagSetsExpr = strings.Join(bits, " | ")
		data.FlagSetsRef = data.FlagSetsConst
	}

	for _, a := range c.Arguments {
		data.Arguments = append(data.Arguments, argumentData{
			Name:        a.Name,
			Description: a.Description,
			Type:        ArgumentTypes[valueOr(a.Type, "string")],
			Optional:    a.Optional,
			Prompt:      a.Prompt,
		})
	}

	if len(c.ArgumentValues) > 0 {
		data.ArgumentValues = predictSet(c.ArgumentValues)
	}

	for _, f := range c.Flags {
		fd := newFlagData(f)
		data.Flags = append(data.Flags, fd)
		if fd.GoType == "time.Duration" {
			data.UsesTime = true
		}
	}

	for _, example := range c.Examples {
		data.Examples = append(data.Examples, exampleFields(example.Description, `appName + " " + c.Name()`, example.Args, example.Output))
	}
	return data
}

func newFlagData(f Flag) flagData {
	typ := valueOr(f.Type, "string")
	def, _ := goDefault(typ, f.Default)

	data := flagData{
		Name:       f.Name,
		Field:      FieldName(f.Name),
		Default:    def,
		Usage:      f.Usage,
		Env:        f.Env,
		Completion: "complete.PredictAnything",
	}
	switch typ {
	case "bool":
		data.GoType, data.Func = "bool", "BoolVar"
		data.Completion = "complete.PredictNothing"
	case "int":
		data.GoType, data.Func = "int", "IntVar"
	case "float":
		data.GoType, data.Func = "float64", "Float64Var"
	case "duration":
		data.GoType, data.Func = "time.Duration", "DurationVar"
	default:
		data.GoType, data.Func = "string", "StringVar"
	}
	if len(f.Values) > 0 {
		data.Completion = predictSet(f.Values)
	}
	return data
}

func predictSet(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "complete.PredictSet(" + strings.Join(quoted, ", ") + ")"
}
