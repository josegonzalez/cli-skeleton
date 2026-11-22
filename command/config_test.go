package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

// configFixture sets up a user config dir and a project dir, and changes
// into a subdirectory of the project.
func configFixture(t *testing.T, user string, project string) (string, string) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("TEST_APP_COUNT", "")
	os.Unsetenv("TEST_APP_COUNT")
	for _, name := range []string{"DEBUG", "TEST_APP_DEBUG"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}

	userPath := filepath.Join(configHome, "test-app", "config.yaml")
	if user != "" {
		if err := os.MkdirAll(filepath.Dir(userPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(userPath, []byte(user), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	projectDir := t.TempDir()
	projectPath := filepath.Join(projectDir, ".test-app.yaml")
	if project != "" {
		if err := os.WriteFile(projectPath, []byte(project), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	nested := filepath.Join(projectDir, "a", "b")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	return userPath, projectPath
}

// countCommand returns a command with an env-bound --count flag.
func countCommand(count *int) *testCommand {
	c := &testCommand{name: "eat", flags: FlagSetClient}
	c.extra = func(f *flag.FlagSet) {
		f.IntVar(count, "count", 1, "number of lollipops")
		f.String("color", "normal", "lollipop color")
		c.EnvFlag(f, "count")
	}
	c.run = func(c *testCommand, _ map[string]Argument) int {
		c.Ui.Output(c.ValueSource("count") + " " + c.ValueSource("color"))
		return 0
	}
	return c
}

func TestConfigPrecedence(t *testing.T) {
	userPath, projectPath := configFixture(t, "count: 2\ncolor: red\n", "eat:\n  count: 3\n")

	var count int
	got := runTestMain(t, []string{"eat"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	if count != 3 {
		t.Fatalf("the project command section should win: count = %d", count)
	}
	if got.stdout != projectPath+" "+userPath+"\n" {
		t.Fatalf("sources = %q", got.stdout)
	}

	t.Setenv("TEST_APP_COUNT", "4")
	got = runTestMain(t, []string{"eat"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if count != 4 || !strings.HasPrefix(got.stdout, "env TEST_APP_COUNT ") {
		t.Fatalf("the environment should override config: count = %d, stdout = %q", count, got.stdout)
	}

	got = runTestMain(t, []string{"eat", "--count", "5"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if count != 5 || !strings.HasPrefix(got.stdout, "flag ") {
		t.Fatalf("flags should override everything: count = %d, stdout = %q", count, got.stdout)
	}
}

func TestConfigIsOptIn(t *testing.T) {
	configFixture(t, "count: 2\n", "")

	var count int
	commands := map[string]*testCommand{"eat": countCommand(&count)}
	runTestMain(t, []string{"eat"}, commands)
	if count != 1 {
		t.Fatalf("config should only be read with WithConfig: count = %d", count)
	}

	got := runTestMain(t, []string{"config"}, commands)
	if got.code != ExitCodeUnknownCommand {
		t.Fatalf("the config command should only exist with WithConfig: %d", got.code)
	}
}

func TestConfigSettings(t *testing.T) {
	configFixture(t, "quiet: true\ndebug: false\n", "")

	commands := map[string]*testCommand{
		"eat": {name: "eat", flags: FlagSetClient, run: func(c *testCommand, _ map[string]Argument) int {
			c.Ui.Info("info")
			if c.IsDebug() {
				c.Ui.Output("debug")
			}
			return 0
		}},
	}

	got := runTestMain(t, []string{"eat"}, commands, WithConfig())
	if got.stderr != "" {
		t.Fatalf("quiet from config should apply before the UI is built: %q", got.stderr)
	}

	got = runTestMain(t, []string{"eat", "--quiet=false"}, commands, WithConfig())
	if got.stderr != "info\n" {
		t.Fatalf("flags should override config settings: %q", got.stderr)
	}

	t.Setenv("DEBUG", "1")
	got = runTestMain(t, []string{"eat"}, commands, WithConfig())
	if got.stdout != "debug\n" {
		t.Fatalf("DEBUG should override debug: false in config: %q", got.stdout)
	}
}

func TestInvalidConfig(t *testing.T) {
	userPath, _ := configFixture(t, "count: [unclosed\n", "")

	var count int
	got := runTestMain(t, []string{"eat"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if got.code != ExitCodeError || !strings.Contains(got.stderr, "Invalid config file "+userPath) {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
}

func TestInvalidConfigValue(t *testing.T) {
	_, projectPath := configFixture(t, "", "count: many\n")

	var count int
	got := runTestMain(t, []string{"eat"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if got.code != ExitCodeUsage || !strings.Contains(got.stderr, projectPath) {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
}

func TestConfigCommand(t *testing.T) {
	userPath, projectPath := configFixture(t, "count: 2\nquiet: false\n", "lollipop eat:\n  color: [red, blue]\n")
	t.Setenv("TEST_APP_COUNT", "7")

	var count int
	commands := map[string]*testCommand{"eat": countCommand(&count)}

	got := runTestMain(t, []string{"config"}, commands, WithConfig())
	if got.code != 0 {
		t.Fatalf("code = %d, stderr = %q", got.code, got.stderr)
	}
	for _, want := range []string{"FILES", userPath, projectPath, "(not found)", "KEY", "count", "env TEST_APP_COUNT", "lollipop-eat.color", "red,blue"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("config output missing %q:\n%s", want, got.stdout)
		}
	}

	got = runTestMain(t, []string{"config", "--json"}, commands, WithConfig())
	var result struct {
		Files  []configFileEntry `json:"files"`
		Values []configEntry     `json:"values"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &result); err != nil {
		t.Fatalf("invalid JSON %q: %v", got.stdout, err)
	}

	found := map[string]configEntry{}
	for _, v := range result.Values {
		found[v.Command+"."+v.Key] = v
	}
	if found[".count"].Value != "2" || found[".count"].Source != userPath {
		t.Errorf("count = %+v", found[".count"])
	}
	if found["eat.count"].Source != "env TEST_APP_COUNT" {
		t.Errorf("eat.count = %+v", found["eat.count"])
	}
	if found["lollipop-eat.color"].Value != "red,blue" {
		t.Errorf("lollipop-eat.color = %+v", found["lollipop-eat.color"])
	}
}

func TestEnvFlagUsage(t *testing.T) {
	m := &Meta{appName: "hello-world"}
	for i := 0; i < 2; i++ {
		f := flag.NewFlagSet("eat", flag.ContinueOnError)
		f.Int("max-count", 1, "the most lollipops")
		m.EnvFlag(f, "max-count")
		m.EnvFlag(f, "missing")

		if got := f.Lookup("max-count").Usage; got != "the most lollipops [env: HELLO_WORLD_MAX_COUNT]" {
			t.Fatalf("usage = %q", got)
		}
	}

	if got := FlagEnv("hello-world", "max-count"); got != "HELLO_WORLD_MAX_COUNT" {
		t.Fatalf("FlagEnv = %q", got)
	}
}

func TestDebugListsValueSources(t *testing.T) {
	userPath, _ := configFixture(t, "count: 2\n", "")

	var count int
	got := runTestMain(t, []string{"eat", "--debug"}, map[string]*testCommand{"eat": countCommand(&count)}, WithConfig())
	if !strings.Contains(got.stderr, "--count=2 (from "+userPath+")") || !strings.Contains(got.stderr, "--debug=true (from flag)") {
		t.Fatalf("stderr = %q", got.stderr)
	}
}
