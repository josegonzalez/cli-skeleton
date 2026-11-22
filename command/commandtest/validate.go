// Package commandtest checks that cli-skeleton apps follow the conventions
// of the Command Line Interface Guidelines (https://clig.dev).
//
// Call ValidateCommands from a test in the app's main package:
//
//	func TestCommands(t *testing.T) {
//		commandtest.ValidateCommands(t, AppName, Commands)
//	}
package commandtest

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/mitchellh/cli"
	flag "github.com/spf13/pflag"
)

// MaxExamples is the most examples a command should show in its help.
// Longer example sets belong in the docs.
const MaxExamples = 5

// MaxPositionalArguments is the most positional arguments a command should
// take, not counting a trailing list. Flags are clearer for anything more.
const MaxPositionalArguments = 4

// reservedShorthands maps the short flags with a conventional meaning to the
// long flag they stand for.
var reservedShorthands = map[string]string{
	"h": "help",
	"v": "version",
	"q": "quiet",
	"d": "debug",
	"f": "force",
	"n": "dry-run",
	"o": "output",
	"a": "all",
	"p": "port",
	"u": "user",
}

// commonCommands are names of widely used commands an app should not take.
var commonCommands = map[string]bool{
	"bash": true, "cat": true, "cd": true, "chmod": true, "convert": true, "cp": true,
	"curl": true, "date": true, "echo": true, "find": true, "git": true, "grep": true,
	"kill": true, "less": true, "ln": true, "ls": true, "make": true, "man": true,
	"mkdir": true, "more": true, "mv": true, "ps": true, "rm": true, "sed": true,
	"sh": true, "sort": true, "ssh": true, "tar": true, "test": true, "time": true,
	"top": true, "touch": true, "vi": true, "which": true,
}

// builtinCommands are added by command.Main.
var builtinCommands = map[string]bool{
	"completion": true,
	"config":     true,
	"docs":       true,
	"examples":   true,
	"help":       true,
}

// secretWords mark flag and env names that look like they hold secrets.
var secretWords = map[string]bool{
	"apikey": true, "credential": true, "credentials": true, "key": true,
	"passphrase": true, "passwd": true, "password": true, "secret": true, "token": true,
}

var (
	appNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	wordPattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	envPattern     = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// ValidateCommands reports every way the app and its commands stray from
// the Command Line Interface Guidelines. It checks that:
//
//   - the app name is lowercase, uses dashes only between words, and is not
//     the name of a common command;
//   - command names are lowercase;
//   - every flag has a long name;
//   - the reserved short flags (-h, -v, -q, -d, -f, -n, -o, -a, -p and -u)
//     keep their conventional meaning, -h is never redefined, and flags with
//     a standard name use their standard short flag;
//   - flags shared by several commands have the same type everywhere;
//   - no two sibling commands have confusingly similar names, such as
//     update and upgrade;
//   - no command shows more than MaxExamples examples;
//   - no command takes more than MaxPositionalArguments positional
//     arguments, besides a trailing list;
//   - flags and environment variables that look like secrets are read from
//     files, such as --password-file;
//   - environment variable names are valid.
func ValidateCommands(t testing.TB, appName string, commandsFunc command.CommandFunc) {
	t.Helper()

	for _, problem := range Problems(appName, commandsFunc) {
		t.Error(problem)
	}
}

// Problems returns the problems ValidateCommands reports.
func Problems(appName string, commandsFunc command.CommandFunc) []string {
	v := &validator{flagTypes: map[string]flagType{}}
	v.checkAppName(appName)

	meta := command.NewMeta(context.Background(), appName, "")
	meta.Ui = cli.NewMockUi()
	commands := commandsFunc(context.Background(), *meta)

	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		v.checkCommandName(name)

		factory := commands[name]
		if factory == nil {
			v.addf("command %q has no factory", name)
			continue
		}
		raw, err := factory()
		if err != nil {
			v.addf("command %q: creating it failed: %v", name, err)
			continue
		}
		c, ok := raw.(command.Command)
		if !ok {
			v.addf("command %q does not implement command.Command; embed command.Meta and add the missing methods", name)
			continue
		}
		v.checkCommand(name, c)
	}

	v.checkSimilarNames(names)
	return v.problems
}

type flagType struct {
	typ     string
	command string
}

type validator struct {
	problems  []string
	flagTypes map[string]flagType
}

func (v *validator) addf(format string, args ...interface{}) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func (v *validator) checkAppName(appName string) {
	if !appNamePattern.MatchString(appName) {
		v.addf("app name %q should be lowercase letters and digits, with dashes only between words", appName)
	}
	if commonCommands[appName] {
		v.addf("app name %q is already a common command; pick a name that will not collide", appName)
	}
}

func (v *validator) checkCommandName(name string) {
	for _, word := range strings.Fields(name) {
		if !wordPattern.MatchString(word) {
			v.addf("command %q: %q should be lowercase letters and digits, with dashes only between words", name, word)
		}
	}
}

func (v *validator) checkCommand(name string, c command.Command) {
	if examples := c.Examples(); len(examples) > MaxExamples {
		v.addf("command %q has %d examples; show at most %d and move the rest to the docs", name, len(examples), MaxExamples)
	}

	positional := 0
	for _, argument := range c.Arguments() {
		if argument.Type != command.ArgumentList {
			positional++
		}
	}
	if positional > MaxPositionalArguments {
		v.addf("command %q takes %d positional arguments; use flags for anything past %d", name, positional, MaxPositionalArguments)
	}

	flags, err := flagSetOf(c)
	if err != nil {
		v.addf("command %q: defining its flags panicked: %v", name, err)
		return
	}
	flags.VisitAll(func(f *flag.Flag) {
		v.checkFlag(name, f)
	})

	if meta, ok := c.(interface{ EnvFlags() map[string]string }); ok {
		for flagName, env := range meta.EnvFlags() {
			if !envPattern.MatchString(env) {
				v.addf("command %q: environment variable %q for --%s should only use uppercase letters, digits and underscores, and not start with a digit", name, env, flagName)
			}
			if looksSecret(env) {
				v.addf("command %q: --%s is bound to %s, but secrets should not be read from environment variables; read them from a file with command.ReadSecret", name, flagName, env)
			}
		}
	}
}

func (v *validator) checkFlag(command string, f *flag.Flag) {
	if len(f.Name) < 2 {
		v.addf("command %q: flag --%s needs a descriptive long name", command, f.Name)
	}

	if f.Shorthand == "h" {
		v.addf("command %q: --%s uses -h, which is reserved for help", command, f.Name)
	} else if want, ok := reservedShorthands[f.Shorthand]; ok && want != f.Name {
		v.addf("command %q: --%s uses -%s, which is reserved for --%s", command, f.Name, f.Shorthand, want)
	}

	for short, long := range reservedShorthands {
		if long == f.Name && f.Shorthand != "" && f.Shorthand != short {
			v.addf("command %q: --%s should use the standard short flag -%s, not -%s", command, f.Name, short, f.Shorthand)
		}
	}

	if previous, ok := v.flagTypes[f.Name]; ok && previous.typ != f.Value.Type() {
		v.addf("flag --%s is a %s in %q but a %s in %q; use the same type everywhere", f.Name, previous.typ, previous.command, f.Value.Type(), command)
	} else if !ok {
		v.flagTypes[f.Name] = flagType{typ: f.Value.Type(), command: command}
	}

	if looksSecret(f.Name) {
		v.addf("command %q: --%s looks like it takes a secret, which leaks into shell history and process lists; accept --%s-file or stdin instead", command, f.Name, f.Name)
	}
}

// flagSetOf returns the flags of c, recovering from panics such as two
// flags sharing a short flag.
func flagSetOf(c command.Command) (flags *flag.FlagSet, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return c.FlagSet(), nil
}

// looksSecret reports whether a flag or env name looks like it holds a
// secret, rather than a path to one.
func looksSecret(name string) bool {
	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(words) == 0 || words[len(words)-1] == "file" || words[len(words)-1] == "path" {
		return false
	}
	for _, word := range words {
		if secretWords[word] {
			return true
		}
	}
	return false
}

// checkSimilarNames reports sibling commands with confusingly similar
// names. Pairs of framework commands are skipped.
func (v *validator) checkSimilarNames(names []string) {
	siblings := map[string][]string{}
	for _, name := range names {
		words := strings.Fields(name)
		parent := strings.Join(words[:len(words)-1], " ")
		siblings[parent] = append(siblings[parent], words[len(words)-1])
	}
	for name := range builtinCommands {
		siblings[""] = append(siblings[""], name)
	}

	for parent, words := range siblings {
		sort.Strings(words)
		for i := 0; i < len(words); i++ {
			for j := i + 1; j < len(words); j++ {
				a, b := words[i], words[j]
				if a == b || (builtinCommands[a] && builtinCommands[b] && parent == "") {
					continue
				}
				if similar(a, b) {
					v.addf("commands %q and %q are easy to confuse; pick names that differ more", strings.TrimSpace(parent+" "+a), strings.TrimSpace(parent+" "+b))
				}
			}
		}
	}
}

// similar reports whether two command names are close enough to be
// confused, such as update and upgrade.
func similar(a string, b string) bool {
	longest := max(len(a), len(b))
	distance := levenshtein(a, b)
	switch {
	case longest >= 6:
		return distance <= 3
	case longest >= 4:
		return distance <= 2
	default:
		return distance <= 1
	}
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a string, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}
