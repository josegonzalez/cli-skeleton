package command

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompletionScripts(t *testing.T) {
	bash := completionScript("bash", "hello-world", "/usr/local/bin/hello-world")
	if !strings.Contains(bash, "complete -o default -C '/usr/local/bin/hello-world' hello-world") {
		t.Errorf("bash = %q", bash)
	}

	zsh := completionScript("zsh", "hello-world", "/opt/it's/hello-world")
	if !strings.HasPrefix(zsh, "#compdef hello-world\n") || !strings.Contains(zsh, `'/opt/it'\''s/hello-world'`) || !strings.Contains(zsh, "COMP_LINE=") {
		t.Errorf("zsh = %q", zsh)
	}

	fish := completionScript("fish", "hello-world", "/usr/local/bin/hello-world")
	if !strings.Contains(fish, "function __complete_hello_world") || !strings.Contains(fish, `complete -f -c hello-world -a "(__complete_hello_world)"`) {
		t.Errorf("fish = %q", fish)
	}
}

func TestCompletionPaths(t *testing.T) {
	data := t.TempDir()
	config := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_CONFIG_HOME", config)

	tests := map[string]string{
		"bash": filepath.Join(data, "bash-completion", "completions", "app"),
		"zsh":  filepath.Join(data, "zsh", "site-functions", "_app"),
		"fish": filepath.Join(config, "fish", "completions", "app.fish"),
	}
	for shell, want := range tests {
		got, err := completionPath(shell, "app")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: %q, want %q", shell, got, want)
		}
	}

	if _, err := completionPath("tcsh", "app"); err == nil {
		t.Error("unsupported shells should fail")
	}
}

func TestCompletionCommandPrintsScript(t *testing.T) {
	got := runTestMain(t, []string{"completion", "fish"}, map[string]*testCommand{"eat": {name: "eat"}})
	if got.code != 0 || !strings.Contains(got.stdout, "complete -f -c test-app") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", got.code, got.stdout, got.stderr)
	}

	t.Setenv("SHELL", "/bin/zsh")
	got = runTestMain(t, []string{"completion"}, map[string]*testCommand{"eat": {name: "eat"}})
	if got.code != 0 || !strings.HasPrefix(got.stdout, "#compdef test-app") {
		t.Fatalf("the shell should default to $SHELL: %q", got.stdout)
	}

	t.Setenv("SHELL", "/bin/tcsh")
	got = runTestMain(t, []string{"completion"}, map[string]*testCommand{"eat": {name: "eat"}})
	if got.code != ExitCodeUsage || !strings.Contains(got.stderr, "bash, zsh, fish") {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}

	got = runTestMain(t, []string{"completion", "tcsh"}, map[string]*testCommand{"eat": {name: "eat"}})
	if got.code != ExitCodeUsage {
		t.Fatalf("an unknown shell should be a usage error: %d", got.code)
	}
}

func TestCompletionInstall(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	rc := filepath.Join(t.TempDir(), ".bashrc")
	t.Setenv("HOME", filepath.Dir(rc))
	commands := map[string]*testCommand{"eat": {name: "eat"}}
	path := filepath.Join(data, "bash-completion", "completions", "test-app")

	got := runTestMain(t, []string{"completion", "bash", "--install"}, commands)
	if got.code != ExitCodeUsage || !strings.Contains(got.stderr, "--force") {
		t.Fatalf("installing without a terminal should need --force: %d %q", got.code, got.stderr)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("nothing should be written without confirmation")
	}

	got = runTestMain(t, []string{"completion", "bash", "--install", "--force"}, commands)
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	data2, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data2), "complete -o default -C") {
		t.Fatalf("script = %q, err = %v", data2, err)
	}
	if _, err := os.Stat(rc); err == nil {
		t.Fatal("shell startup files should never be changed")
	}

	got = runTestMain(t, []string{"completion", "bash", "--uninstall"}, commands)
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("the script should be removed")
	}

	got = runTestMain(t, []string{"--autocomplete-install", "--force"}, commands)
	if got.code != ExitCodeUsage && got.code != 0 {
		t.Fatalf("--autocomplete-install should go through the completion command: %d %q", got.code, got.stderr)
	}
}

func TestCompletionInstallDeclined(t *testing.T) {
	fakeTerminal(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)

	got := runTestMainWithInput(t, "n\n", []string{"completion", "bash", "--install"}, map[string]*testCommand{"eat": {name: "eat"}})
	if got.code != 0 || !strings.Contains(got.stderr, "Write the bash completion script to") || !strings.Contains(got.stderr, "Nothing was changed.") {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	if _, err := os.Stat(filepath.Join(data, "bash-completion", "completions", "test-app")); err == nil {
		t.Fatal("declining should not write the script")
	}
}

func TestRewriteAutocompleteArgs(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: nil, want: nil},
		{args: []string{"eat"}, want: []string{"eat"}},
		{args: []string{"-autocomplete-install"}, want: []string{"completion", "--install"}},
		{args: []string{"--autocomplete-uninstall", "--force"}, want: []string{"completion", "--uninstall", "--force"}},
	}
	for _, tt := range tests {
		if got := rewriteAutocompleteArgs(tt.args); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("rewriteAutocompleteArgs(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMainAnswersCompletionRequests(t *testing.T) {
	t.Setenv("COMP_LINE", "test-app ea")
	t.Setenv("COMP_POINT", "")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	got := runTestMain(t, []string{"test-app", "ea", "test-app"}, map[string]*testCommand{
		"eat":   {name: "eat"},
		"drink": {name: "drink"},
	})
	w.Close()
	os.Stdout = original

	out, _ := io.ReadAll(r)
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	if strings.TrimSpace(string(out)) != "eat" {
		t.Fatalf("completions = %q", out)
	}
	if strings.Contains(got.stderr, "Unknown command") {
		t.Fatal("completion requests should skip command checks")
	}
}
