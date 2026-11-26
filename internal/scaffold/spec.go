// Package scaffold generates the source of a new cli-skeleton app from a
// Spec describing its commands.
package scaffold

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Spec describes an app to generate.
type Spec struct {
	// Name is the name of the app and its binary.
	Name string `yaml:"name"`

	// Module is the Go module path. It defaults to Name.
	Module string `yaml:"module,omitempty"`

	// Description is shown at the top of the app help.
	Description string `yaml:"description,omitempty"`

	// Examples are shown in the app help.
	Examples []AppExample `yaml:"examples,omitempty"`

	// UI is the UI the app uses: default, zerolog or human-zerolog.
	UI string `yaml:"ui,omitempty"`

	// Config reads flag defaults from config files with command.WithConfig.
	Config bool `yaml:"config,omitempty"`

	DocsURL    string `yaml:"docs_url,omitempty"`
	SupportURL string `yaml:"support_url,omitempty"`
	IssueURL   string `yaml:"issue_url,omitempty"`

	// Aliases maps an alias to the command it runs.
	Aliases map[string]string `yaml:"aliases,omitempty"`

	// GlobalFlags are accepted by every command, before or after the
	// command name.
	GlobalFlags []Flag `yaml:"global_flags,omitempty"`

	// Commands are the commands of the app.
	Commands []Command `yaml:"commands"`
}

// AppExample is an example shown in the app help.
type AppExample struct {
	Description string `yaml:"description"`

	// Command is what follows the app name, such as `eat --count 2`.
	Command string `yaml:"command,omitempty"`

	Output string `yaml:"output,omitempty"`
}

// Command describes a command of the app.
type Command struct {
	// Name is the command path, such as `lollipop eat`.
	Name string `yaml:"name"`

	Synopsis    string `yaml:"synopsis"`
	Description string `yaml:"description,omitempty"`

	// Deprecated hides the command and warns when it is used.
	Deprecated string `yaml:"deprecated,omitempty"`

	// FlagSets are the framework flag sets the command uses, such as
	// client, json or force. They default to client.
	FlagSets []string `yaml:"flag_sets,omitempty"`

	Arguments []Argument `yaml:"arguments,omitempty"`

	// ArgumentValues are offered when completing the arguments.
	ArgumentValues []string `yaml:"argument_values,omitempty"`

	Flags    []Flag    `yaml:"flags,omitempty"`
	Examples []Example `yaml:"examples,omitempty"`
}

// Argument describes a positional argument of a command.
type Argument struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	// Type is one of string, int, bool, float, duration, file or list.
	// It defaults to string.
	Type string `yaml:"type,omitempty"`

	Optional bool `yaml:"optional,omitempty"`

	// Prompt is asked for a missing required argument on a terminal.
	Prompt string `yaml:"prompt,omitempty"`
}

// Flag describes a flag of a command, or a global flag.
type Flag struct {
	Name string `yaml:"name"`

	// Type is one of bool, int, string, float or duration. It defaults to
	// string.
	Type string `yaml:"type,omitempty"`

	Default string `yaml:"default,omitempty"`
	Usage   string `yaml:"usage"`

	// Env lets `<APP>_<FLAG>` set the flag default.
	Env bool `yaml:"env,omitempty"`

	// Values are offered when completing the flag value.
	Values []string `yaml:"values,omitempty"`
}

// Example is an example shown in the help of a command.
type Example struct {
	Description string `yaml:"description"`

	// Args is what follows the command name, such as `--count 2`.
	Args string `yaml:"args,omitempty"`

	Output string `yaml:"output,omitempty"`
}

// UI names accepted by Spec.UI.
const (
	UIDefault      = "default"
	UIZerolog      = "zerolog"
	UIHumanZerolog = "human-zerolog"
)

// UIs lists the accepted UI names.
var UIs = []string{UIDefault, UIZerolog, UIHumanZerolog}

// ArgumentTypes maps argument types to their command.Argument type.
var ArgumentTypes = map[string]string{
	"string":   "command.ArgumentString",
	"int":      "command.ArgumentInt",
	"bool":     "command.ArgumentBool",
	"float":    "command.ArgumentFloat",
	"duration": "command.ArgumentDuration",
	"file":     "command.ArgumentFile",
	"list":     "command.ArgumentList",
}

// FlagTypes lists the accepted flag types.
var FlagTypes = []string{"bool", "int", "string", "float", "duration"}

// FlagSets maps flag set names to their command.FlagSet bit.
var FlagSets = map[string]string{
	"none":    "command.FlagSetNone",
	"client":  "command.FlagSetClient",
	"json":    "command.FlagSetJSON",
	"plain":   "command.FlagSetPlain",
	"force":   "command.FlagSetForce",
	"dry-run": "command.FlagSetDryRun",
	"confirm": "command.FlagSetConfirm",
	"timeout": "command.FlagSetTimeout",
	"output":  "command.FlagSetOutput",
	"all":     "command.FlagSetAll",
	"port":    "command.FlagSetPort",
	"user":    "command.FlagSetUser",
	"default": "command.FlagSetDefault",
}

// StandardFlagSets lists the opt-in standard flag sets, in the order the
// README documents them.
var StandardFlagSets = []string{"json", "plain", "force", "dry-run", "confirm", "timeout", "output", "all", "port", "user"}

// frameworkFlags are added to every command by the framework.
var frameworkFlags = []string{"help", "no-color", "no-input", "quiet", "debug", "no-pager"}

// reservedFlags returns the flags a command with the given flag sets gets
// from the framework, so it cannot define them itself. Global flags pass
// every flag set, since they are added to every command.
func reservedFlags(flagSets []string) map[string]bool {
	reserved := map[string]bool{}
	for _, name := range frameworkFlags {
		reserved[name] = true
	}
	for _, name := range StandardFlagSets {
		if contains(flagSets, name) || contains(flagSets, "default") {
			reserved[name] = true
		}
	}
	return reserved
}

// reservedCommands are added by the generated app itself.
var reservedCommands = map[string]bool{
	"version": true,
}

// MaxExamples and MaxPositionalArguments match the limits checked by
// commandtest.ValidateCommands, so generated apps pass their own tests.
const (
	MaxExamples            = 5
	MaxPositionalArguments = 4
)

var (
	namePattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	modulePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~/-]*$`)
)

// ParseSpec reads a YAML spec.
func ParseSpec(r io.Reader) (*Spec, error) {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	var spec Spec
	if err := decoder.Decode(&spec); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the spec is empty")
		}
		return nil, err
	}
	return &spec, nil
}

// Marshal returns the spec as YAML.
func (s *Spec) Marshal() ([]byte, error) {
	return yaml.Marshal(s)
}

// ModulePath returns the module path, defaulting to the app name.
func (s *Spec) ModulePath() string {
	if s.Module != "" {
		return s.Module
	}
	return s.Name
}

// Validate reports every problem with the spec.
func (s *Spec) Validate() error {
	v := &validator{}

	switch {
	case s.Name == "":
		v.addf("name is required")
	case !namePattern.MatchString(s.Name):
		v.addf("name %q should be lowercase letters and digits, with dashes only between words", s.Name)
	}
	if s.Module != "" && !modulePattern.MatchString(s.Module) {
		v.addf("module %q is not a valid module path", s.Module)
	}
	if s.UI != "" && !contains(UIs, s.UI) {
		v.addf("ui %q should be one of %s", s.UI, strings.Join(UIs, ", "))
	}
	for i, example := range s.Examples {
		if example.Description == "" {
			v.addf("examples[%d]: description is required", i)
		}
	}

	fields := map[string]string{}
	globalNames := map[string]bool{}
	for i, f := range s.GlobalFlags {
		prefix := fmt.Sprintf("global_flags[%d]", i)
		if f.Name != "" {
			prefix = fmt.Sprintf("global flag --%s", f.Name)
		}
		if globalNames[f.Name] {
			v.addf("%s is defined more than once", prefix)
			continue
		}
		v.checkFlag(prefix, f, fields, reservedFlags([]string{"default"}))
		if f.Env {
			v.addf("%s: env is only supported on command flags", prefix)
		}
		globalNames[f.Name] = true
	}

	if len(s.Commands) == 0 {
		v.addf("at least one command is required")
	}
	names := map[string]bool{}
	types := map[string]string{"GlobalFlagCommand": "the global flags"}
	files := map[string]string{}
	for i, c := range s.Commands {
		prefix := fmt.Sprintf("commands[%d]", i)
		if c.Name != "" {
			prefix = fmt.Sprintf("command %q", c.Name)
		}
		v.checkCommand(prefix, c, globalNames, fields)

		if c.Name == "" {
			continue
		}
		if names[c.Name] {
			v.addf("%s is defined more than once", prefix)
			continue
		}
		names[c.Name] = true

		typ := TypeName(c.Name)
		if other, ok := types[typ]; ok {
			v.addf("%s and %s would both be the type %s; rename one of them", prefix, other, typ)
		}
		types[typ] = prefix

		file := FileName(c.Name)
		if other, ok := files[file]; ok {
			v.addf("%s and %s would both be written to %s; rename one of them", prefix, other, file)
		}
		files[file] = prefix
	}

	aliases := make([]string, 0, len(s.Aliases))
	for alias := range s.Aliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		target := s.Aliases[alias]
		if !validCommandName(alias) {
			v.addf("alias %q should be lowercase words of letters and digits, with dashes only between words", alias)
		}
		if names[alias] {
			v.addf("alias %q is already a command", alias)
		}
		if !names[target] {
			v.addf("alias %q points to %q, which is not a command", alias, target)
		}
	}

	return v.err()
}

type validator struct {
	problems []string
}

func (v *validator) addf(format string, args ...interface{}) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func (v *validator) err() error {
	if len(v.problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: v.problems}
}

// ValidationError lists the problems found by Spec.Validate.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	if len(e.Problems) == 1 {
		return "invalid spec: " + e.Problems[0]
	}
	return "invalid spec:\n  " + strings.Join(e.Problems, "\n  ")
}

// validCommandName reports whether every word of name follows the naming
// rules of commandtest.ValidateCommands.
func validCommandName(name string) bool {
	words := strings.Fields(name)
	if len(words) == 0 || strings.Join(words, " ") != name {
		return false
	}
	for _, word := range words {
		if !namePattern.MatchString(word) {
			return false
		}
	}
	return true
}

func (v *validator) checkCommand(prefix string, c Command, globalNames map[string]bool, globalFields map[string]string) {
	switch {
	case c.Name == "":
		v.addf("%s: name is required", prefix)
	case !validCommandName(c.Name):
		v.addf("%s: every word should be lowercase letters and digits, with dashes only between words, separated by single spaces", prefix)
	case reservedCommands[c.Name]:
		v.addf("%s is added to every app, so it cannot be defined", prefix)
	}
	if c.Synopsis == "" {
		v.addf("%s: synopsis is required", prefix)
	}

	for _, name := range c.FlagSets {
		if _, ok := FlagSets[name]; !ok {
			v.addf("%s: unknown flag set %q", prefix, name)
		}
	}

	argNames := map[string]bool{}
	positional := 0
	reachedOptional := false
	reachedList := false
	for i, a := range c.Arguments {
		p := fmt.Sprintf("%s: arguments[%d]", prefix, i)
		switch {
		case a.Name == "":
			v.addf("%s: name is required", p)
		case strings.ContainsAny(a.Name, " \t\n<>[]"):
			v.addf("%s: name %q cannot contain spaces or brackets", p, a.Name)
		case argNames[a.Name]:
			v.addf("%s: argument %q is defined more than once", p, a.Name)
		}
		argNames[a.Name] = true

		if a.Description == "" {
			v.addf("%s: description is required", p)
		}
		typ := valueOr(a.Type, "string")
		if _, ok := ArgumentTypes[typ]; !ok {
			v.addf("%s: unknown type %q", p, a.Type)
		}
		if typ != "list" {
			positional++
		}

		if reachedList {
			v.addf("%s: the list argument must be the last argument", p)
		}
		if typ == "list" {
			reachedList = true
		}
		if reachedOptional && !a.Optional {
			v.addf("%s: required arguments must come before optional ones", p)
		}
		if a.Optional {
			reachedOptional = true
		}
	}
	if positional > MaxPositionalArguments {
		v.addf("%s: takes %d positional arguments; use flags for anything past %d", prefix, positional, MaxPositionalArguments)
	}

	fields := map[string]string{}
	for k, val := range globalFields {
		fields[k] = val
	}
	reserved := reservedFlags(c.FlagSets)
	flagNames := map[string]bool{}
	for i, f := range c.Flags {
		p := fmt.Sprintf("%s: flags[%d]", prefix, i)
		if f.Name != "" {
			p = fmt.Sprintf("%s: flag --%s", prefix, f.Name)
		}
		if flagNames[f.Name] {
			v.addf("%s is defined more than once", p)
			continue
		}
		flagNames[f.Name] = true
		if globalNames[f.Name] {
			v.addf("%s is already a global flag", p)
			continue
		}
		v.checkFlag(p, f, fields, reserved)
	}

	if len(c.Examples) > MaxExamples {
		v.addf("%s: has %d examples; show at most %d", prefix, len(c.Examples), MaxExamples)
	}
	for i, example := range c.Examples {
		if example.Description == "" {
			v.addf("%s: examples[%d]: description is required", prefix, i)
		}
	}
}

func (v *validator) checkFlag(prefix string, f Flag, fields map[string]string, reserved map[string]bool) {
	switch {
	case f.Name == "":
		v.addf("%s: name is required", prefix)
		return
	case !namePattern.MatchString(f.Name):
		v.addf("%s: the name should be lowercase letters and digits, with dashes only between words", prefix)
		return
	case len(f.Name) < 2:
		v.addf("%s: the name should be at least two characters long", prefix)
	case reserved[f.Name]:
		v.addf("%s is defined by the framework; pick another name", prefix)
	}

	field := FieldName(f.Name)
	if other, ok := fields[field]; ok {
		v.addf("%s and %s would both be the field %s; rename one of them", prefix, other, field)
	}
	fields[field] = prefix

	if f.Usage == "" {
		v.addf("%s: usage is required", prefix)
	}

	typ := valueOr(f.Type, "string")
	if !contains(FlagTypes, typ) {
		v.addf("%s: unknown type %q", prefix, f.Type)
		return
	}
	if _, err := goDefault(typ, f.Default); err != nil {
		v.addf("%s: %v", prefix, err)
	}
}

// goDefault returns the Go literal for the default value of a flag.
func goDefault(typ string, value string) (string, error) {
	switch typ {
	case "bool":
		if value == "" {
			return "false", nil
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return "", fmt.Errorf("default %q is not a bool", value)
		}
		return strconv.FormatBool(b), nil
	case "int":
		if value == "" {
			return "0", nil
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			return "", fmt.Errorf("default %q is not an int", value)
		}
		return strconv.Itoa(i), nil
	case "float":
		if value == "" {
			return "0", nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", fmt.Errorf("default %q is not a float", value)
		}
		return strconv.FormatFloat(f, 'g', -1, 64), nil
	case "duration":
		if value == "" {
			return "0", nil
		}
		d, err := time.ParseDuration(value)
		if err != nil {
			return "", fmt.Errorf("default %q is not a duration, such as 30s or 5m", value)
		}
		return durationLiteral(d), nil
	default:
		return strconv.Quote(value), nil
	}
}

// durationLiteral returns d as a Go expression, such as `3 * time.Second`.
func durationLiteral(d time.Duration) string {
	if d == 0 {
		return "0"
	}
	units := []struct {
		unit time.Duration
		name string
	}{
		{time.Hour, "time.Hour"},
		{time.Minute, "time.Minute"},
		{time.Second, "time.Second"},
		{time.Millisecond, "time.Millisecond"},
		{time.Microsecond, "time.Microsecond"},
		{time.Nanosecond, "time.Nanosecond"},
	}
	for _, u := range units {
		if d%u.unit == 0 {
			n := d / u.unit
			if n == 1 {
				return u.name
			}
			if n == -1 {
				return "-" + u.name
			}
			return fmt.Sprintf("%d * %s", n, u.name)
		}
	}
	return strconv.FormatInt(int64(d), 10)
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
