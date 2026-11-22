package command

import (
	"os"
	"testing"
)

// clearColorEnv unsets every env var that affects color detection for the
// duration of the test.
func clearColorEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{EnvCLINoColor, EnvForceColor, "TERM", "HELLO_WORLD_NO_COLOR"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
}

func TestEnvPrefix(t *testing.T) {
	tests := map[string]string{
		"hello-world": "HELLO_WORLD",
		"app":         "APP",
		"my.app v2":   "MY_APP_V2",
		"9lives":      "_9LIVES",
		"":            "",
	}
	for in, want := range tests {
		if got := EnvPrefix(in); got != want {
			t.Errorf("EnvPrefix(%q) = %q, want %q", in, got, want)
		}
	}

	if got := AppEnv("hello-world", "NO_COLOR"); got != "HELLO_WORLD_NO_COLOR" {
		t.Errorf("AppEnv = %q", got)
	}
	if got := AppEnv("", "NO_COLOR"); got != "" {
		t.Errorf("AppEnv with empty app = %q", got)
	}
}

func TestColorEnabled(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		isTerminal bool
		flag       bool
		want       bool
	}{
		{name: "terminal", isTerminal: true, want: true},
		{name: "not a terminal", isTerminal: false, want: false},
		{name: "flag", isTerminal: true, flag: true, want: false},
		{name: "NO_COLOR", isTerminal: true, env: map[string]string{"NO_COLOR": "1"}, want: false},
		{name: "NO_COLOR any value", isTerminal: true, env: map[string]string{"NO_COLOR": "false"}, want: false},
		{name: "app NO_COLOR", isTerminal: true, env: map[string]string{"HELLO_WORLD_NO_COLOR": "1"}, want: false},
		{name: "TERM dumb", isTerminal: true, env: map[string]string{"TERM": "dumb"}, want: false},
		{name: "FORCE_COLOR", isTerminal: false, env: map[string]string{"FORCE_COLOR": "1"}, want: true},
		{name: "FORCE_COLOR overrides TERM dumb", isTerminal: false, env: map[string]string{"FORCE_COLOR": "1", "TERM": "dumb"}, want: true},
		{name: "FORCE_COLOR=0", isTerminal: false, env: map[string]string{"FORCE_COLOR": "0"}, want: false},
		{name: "NO_COLOR beats FORCE_COLOR", isTerminal: true, env: map[string]string{"FORCE_COLOR": "1", "NO_COLOR": "1"}, want: false},
		{name: "flag beats FORCE_COLOR", isTerminal: true, flag: true, env: map[string]string{"FORCE_COLOR": "1"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearColorEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := colorEnabled(tt.isTerminal, "hello-world", tt.flag); got != tt.want {
				t.Fatalf("colorEnabled = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColorEnabledPipeIsNotATerminal(t *testing.T) {
	clearColorEnv(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if IsTerminal(w) {
		t.Fatal("pipe reported as terminal")
	}
	if ColorEnabled(w, "hello-world", false) {
		t.Fatal("color enabled for a pipe")
	}
	if IsTerminal(nil) {
		t.Fatal("nil file reported as terminal")
	}
}
