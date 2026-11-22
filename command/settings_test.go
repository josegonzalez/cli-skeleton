package command

import (
	"errors"
	"os"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

func testGlobalFlags(f *flag.FlagSet) {
	f.Bool("global", false, "a bool global flag")
	f.StringP("global-string", "g", "", "a string global flag")
}

func TestPrescanSettings(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Settings
	}{
		{name: "none", args: []string{"eat"}, want: Settings{}},
		{name: "long flags", args: []string{"eat", "--no-color", "--no-input", "--quiet", "--debug", "--no-pager"}, want: Settings{NoColor: true, NoInput: true, Quiet: true, Debug: true, NoPager: true}},
		{name: "before the subcommand", args: []string{"--quiet", "eat"}, want: Settings{Quiet: true}},
		{name: "explicit value", args: []string{"eat", "--quiet=true", "--debug=false"}, want: Settings{Quiet: true}},
		{name: "short flags", args: []string{"eat", "-q", "-d"}, want: Settings{Quiet: true, Debug: true}},
		{name: "short cluster", args: []string{"eat", "-qd"}, want: Settings{Quiet: true, Debug: true}},
		{name: "cluster with unknown short", args: []string{"eat", "-fq"}, want: Settings{Quiet: true}},
		{name: "single-dash long flag is a cluster", args: []string{"eat", "-no-color"}, want: Settings{}},
		{name: "stops at --", args: []string{"eat", "--", "--quiet"}, want: Settings{}},
		{name: "value of a known flag is skipped", args: []string{"eat", "--global-string", "--quiet"}, want: Settings{}},
		{name: "value of a known short flag is skipped", args: []string{"eat", "-g", "--quiet"}, want: Settings{}},
		{name: "attached short value", args: []string{"eat", "-gq", "--debug"}, want: Settings{Debug: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &Settings{}
			prescanSettings(got, tt.args, globalFlagSet(testGlobalFlags))
			if *got != tt.want {
				t.Fatalf("prescanSettings(%v) = %+v, want %+v", tt.args, *got, tt.want)
			}
		})
	}
}

func TestEnvSettings(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Settings
	}{
		{name: "none", want: Settings{}},
		{name: "DEBUG", env: map[string]string{"DEBUG": "1"}, want: Settings{Debug: true}},
		{name: "DEBUG=0", env: map[string]string{"DEBUG": "0"}, want: Settings{}},
		{name: "DEBUG=false", env: map[string]string{"DEBUG": "false"}, want: Settings{}},
		{name: "DEBUG with any value", env: map[string]string{"DEBUG": "*"}, want: Settings{Debug: true}},
		{name: "app DEBUG", env: map[string]string{"HELLO_WORLD_DEBUG": "true"}, want: Settings{Debug: true}},
		{name: "app VERBOSE", env: map[string]string{"HELLO_WORLD_VERBOSE": "1"}, want: Settings{Verbose: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range []string{"DEBUG", "HELLO_WORLD_DEBUG", "HELLO_WORLD_VERBOSE"} {
				t.Setenv(name, "")
				os.Unsetenv(name)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := &Settings{}
			applyEnvSettings(got, "hello-world")
			if *got != tt.want {
				t.Fatalf("applyEnvSettings = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestArgsOverrideEnvSettings(t *testing.T) {
	t.Setenv("DEBUG", "1")
	got := resolveSettings("hello-world", []string{"eat", "--debug=false"}, globalFlagSet(nil), nil)
	if got.Debug {
		t.Fatal("--debug=false should override DEBUG")
	}
}

func TestFlagSetKeepsResolvedSettings(t *testing.T) {
	m := &Meta{settings: &Settings{Quiet: true}}

	f := m.FlagSet("eat", FlagSetClient)
	if !m.IsQuiet() {
		t.Fatal("defining flags reset a resolved setting")
	}
	if got := f.Lookup("quiet").DefValue; got != "false" {
		t.Fatalf("quiet default = %q, want false", got)
	}

	if err := f.Parse([]string{"--debug", "--quiet=false"}); err != nil {
		t.Fatal(err)
	}
	if !m.IsDebug() || m.IsQuiet() {
		t.Fatalf("parsed flags not stored in settings: %+v", *m.settings)
	}
	if !m.IsVerbose() {
		t.Fatal("debug should imply verbose")
	}
}

func TestFlagSetNoneHasNoClientFlags(t *testing.T) {
	m := &Meta{}
	f := m.FlagSet("eat", FlagSetNone)
	if f.Lookup("quiet") != nil {
		t.Fatal("FlagSetNone should not add framework flags")
	}
	if m.AutocompleteFlags(FlagSetNone) != nil {
		t.Fatal("FlagSetNone should not add framework flag completions")
	}
}

func TestSettingsUiQuiet(t *testing.T) {
	inner, stdout, stderr := newTestTerminalUi("", false)
	ui := newSettingsUi(inner, &Settings{Quiet: true}, nil)

	ui.Info("info")
	ui.Output("output")
	ui.Warn("warn")
	ui.Error("error")

	if stdout.String() != "output\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
	if stderr.String() != "warn\nerror\n" {
		t.Errorf("stderr = %q, info should be dropped", stderr.String())
	}
}

func TestSettingsUiRefusesPrompts(t *testing.T) {
	inner, _, _ := newTestTerminalUi("answer\n", false)

	ui := newSettingsUi(inner, &Settings{NoInput: true}, os.Stdin)
	_, err := ui.Ask("Name?")
	if !errors.Is(err, ErrNoInput) || !strings.Contains(err.Error(), "--no-input") {
		t.Fatalf("expected a --no-input error, got %v", err)
	}

	ui = newSettingsUi(inner, &Settings{}, strings.NewReader("answer\n"))
	_, err = ui.AskSecret("Password:")
	if !errors.Is(err, ErrNoInput) || !strings.Contains(err.Error(), "not a terminal") {
		t.Fatalf("expected a terminal error, got %v", err)
	}
}
