package scaffold

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validSpec() *Spec {
	return &Spec{Name: "app", Commands: []Command{{Name: "run", Synopsis: "Run it"}}}
}

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		edit func(s *Spec)
		want string
	}{
		"valid":             {func(s *Spec) {}, ""},
		"missing name":      {func(s *Spec) { s.Name = "" }, "name is required"},
		"uppercase name":    {func(s *Spec) { s.Name = "App" }, "should be lowercase"},
		"bad module":        {func(s *Spec) { s.Module = "has space" }, "not a valid module path"},
		"unknown ui":        {func(s *Spec) { s.UI = "fancy" }, `ui "fancy"`},
		"app example":       {func(s *Spec) { s.Examples = []AppExample{{Command: "run"}} }, "examples[0]: description is required"},
		"no commands":       {func(s *Spec) { s.Commands = nil }, "at least one command"},
		"command name":      {func(s *Spec) { s.Commands[0].Name = "Run" }, "every word should be lowercase"},
		"double space":      {func(s *Spec) { s.Commands[0].Name = "app  run" }, "every word should be lowercase"},
		"reserved command":  {func(s *Spec) { s.Commands[0].Name = "version" }, "added to every app"},
		"missing synopsis":  {func(s *Spec) { s.Commands[0].Synopsis = "" }, "synopsis is required"},
		"duplicate command": {func(s *Spec) { s.Commands = append(s.Commands, s.Commands[0]) }, "defined more than once"},
		"same type": {func(s *Spec) {
			s.Commands = append(s.Commands, Command{Name: "run-it", Synopsis: "x"}, Command{Name: "run it", Synopsis: "x"})
		}, "would both be the type RunItCommand"},
		"global flag type":     {func(s *Spec) { s.Commands[0].Name = "global flag" }, "would both be the type GlobalFlagCommand"},
		"unknown flag set":     {func(s *Spec) { s.Commands[0].FlagSets = []string{"colour"} }, `unknown flag set "colour"`},
		"argument name":        {func(s *Spec) { s.Commands[0].Arguments = []Argument{{Name: "a b", Description: "x"}} }, "cannot contain spaces"},
		"argument description": {func(s *Spec) { s.Commands[0].Arguments = []Argument{{Name: "a"}} }, "description is required"},
		"argument type":        {func(s *Spec) { s.Commands[0].Arguments = []Argument{{Name: "a", Description: "x", Type: "map"}} }, `unknown type "map"`},
		"duplicate argument": {func(s *Spec) {
			s.Commands[0].Arguments = []Argument{{Name: "a", Description: "x"}, {Name: "a", Description: "x"}}
		}, "defined more than once"},
		"list not last": {func(s *Spec) {
			s.Commands[0].Arguments = []Argument{{Name: "a", Description: "x", Type: "list"}, {Name: "b", Description: "x"}}
		}, "must be the last argument"},
		"optional before needed": {func(s *Spec) {
			s.Commands[0].Arguments = []Argument{{Name: "a", Description: "x", Optional: true}, {Name: "b", Description: "x"}}
		}, "before optional ones"},
		"too many arguments": {func(s *Spec) {
			for _, name := range []string{"a", "b", "c", "d", "e"} {
				s.Commands[0].Arguments = append(s.Commands[0].Arguments, Argument{Name: name, Description: "x"})
			}
		}, "takes 5 positional arguments"},
		"too many examples": {func(s *Spec) {
			for i := 0; i < 6; i++ {
				s.Commands[0].Examples = append(s.Commands[0].Examples, Example{Description: "x"})
			}
		}, "has 6 examples"},
		"example description": {func(s *Spec) { s.Commands[0].Examples = []Example{{Args: "x"}} }, "examples[0]: description is required"},
		"flag name":           {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "Count", Usage: "x"}} }, "should be lowercase"},
		"short flag name":     {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "c", Usage: "x"}} }, "at least two characters"},
		"framework flag":      {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "quiet", Usage: "x"}} }, "defined by the framework"},
		"standard flag in use": {func(s *Spec) {
			s.Commands[0].FlagSets = []string{"client", "json"}
			s.Commands[0].Flags = []Flag{{Name: "json", Type: "bool", Usage: "x"}}
		}, "defined by the framework"},
		"flag usage":       {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "count"}} }, "usage is required"},
		"flag type":        {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "count", Type: "uint", Usage: "x"}} }, `unknown type "uint"`},
		"bad int default":  {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "count", Type: "int", Default: "two", Usage: "x"}} }, "is not an int"},
		"bad bool default": {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "on", Type: "bool", Default: "maybe", Usage: "x"}} }, "is not a bool"},
		"bad duration": {func(s *Spec) {
			s.Commands[0].Flags = []Flag{{Name: "wait", Type: "duration", Default: "5", Usage: "x"}}
		}, "is not a duration"},
		"bad float default": {func(s *Spec) {
			s.Commands[0].Flags = []Flag{{Name: "ratio", Type: "float", Default: "half", Usage: "x"}}
		}, "is not a float"},
		"duplicate flag": {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "count", Usage: "x"}, {Name: "count", Usage: "x"}} }, "defined more than once"},
		"same field":     {func(s *Spec) { s.Commands[0].Flags = []Flag{{Name: "a-1b", Usage: "x"}, {Name: "a1b", Usage: "x"}} }, "would both be the field a1b"},
		"global flag twice": {func(s *Spec) {
			s.GlobalFlags = []Flag{{Name: "region", Usage: "x"}}
			s.Commands[0].Flags = []Flag{{Name: "region", Usage: "x"}}
		}, "already a global flag"},
		"global flag env":           {func(s *Spec) { s.GlobalFlags = []Flag{{Name: "region", Usage: "x", Env: true}} }, "env is only supported on command flags"},
		"global flag standard name": {func(s *Spec) { s.GlobalFlags = []Flag{{Name: "json", Usage: "x"}} }, "defined by the framework"},
		"alias to nothing":          {func(s *Spec) { s.Aliases = map[string]string{"r": "start"} }, `points to "start"`},
		"alias of a command":        {func(s *Spec) { s.Aliases = map[string]string{"run": "run"} }, "already a command"},
		"alias name":                {func(s *Spec) { s.Aliases = map[string]string{"R": "run"} }, `alias "R" should be lowercase`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validSpec()
			tt.edit(spec)
			err := spec.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestValidateReportsEveryProblem(t *testing.T) {
	err := (&Spec{Commands: []Command{{Name: "run"}}}).Validate()
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v, want a ValidationError", err)
	}
	if len(verr.Problems) != 2 {
		t.Errorf("problems = %q, want the missing name and synopsis", verr.Problems)
	}
}

func TestParseSpec(t *testing.T) {
	spec, err := ParseSpec(strings.NewReader(`
name: app
commands:
  - name: run
    synopsis: Run it
    flags:
      - name: count
        type: int
        default: 2
        usage: how many
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := spec.Commands[0].Flags[0].Default; got != "2" {
		t.Errorf("an unquoted default should be read as a string, got %q", got)
	}

	if _, err := ParseSpec(strings.NewReader("name: app\nbogus: true\n")); err == nil {
		t.Error("expected an error for an unknown field")
	}
	if _, err := ParseSpec(strings.NewReader("")); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want an error for an empty spec", err)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	for name, spec := range exampleSpecs(t) {
		contents, err := spec.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseSpec(bytes.NewReader(contents))
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, contents)
		}
		if !reflect.DeepEqual(parsed, spec) {
			t.Errorf("%s: the spec changed after a round trip:\n%s", name, contents)
		}
	}
}

func TestNames(t *testing.T) {
	tests := []struct {
		command, typ, file string
	}{
		{"eat", "EatCommand", "eat.go"},
		{"lollipop eat", "LollipopEatCommand", "lollipop-eat.go"},
		{"app dry-run", "AppDryRunCommand", "app-dry-run.go"},
		{"server linux", "ServerLinuxCommand", "server-linux.go"},
		{"unit test", "UnitTestCommand", "unit-test.go"},
	}
	for _, tt := range tests {
		if got := TypeName(tt.command); got != tt.typ {
			t.Errorf("TypeName(%q) = %q, want %q", tt.command, got, tt.typ)
		}
		if got := FileName(tt.command); got != tt.file {
			t.Errorf("FileName(%q) = %q, want %q", tt.command, got, tt.file)
		}
	}

	fields := map[string]string{
		"count":         "count",
		"global-string": "globalString",
		"type":          "typeValue",
		"meta":          "metaValue",
		"dry-run-mode":  "dryRunMode",
	}
	for flag, want := range fields {
		if got := FieldName(flag); got != want {
			t.Errorf("FieldName(%q) = %q, want %q", flag, got, want)
		}
	}
}

func TestDurationLiteral(t *testing.T) {
	tests := map[time.Duration]string{
		0:                       "0",
		time.Second:             "time.Second",
		3 * time.Second:         "3 * time.Second",
		90 * time.Second:        "90 * time.Second",
		2 * time.Hour:           "2 * time.Hour",
		1500 * time.Millisecond: "1500 * time.Millisecond",
		-time.Minute:            "-time.Minute",
	}
	for d, want := range tests {
		if got := durationLiteral(d); got != want {
			t.Errorf("durationLiteral(%s) = %q, want %q", d, got, want)
		}
	}
}
