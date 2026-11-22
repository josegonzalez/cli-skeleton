package command

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// fakeStdoutTerminal makes stdout count as a terminal for the test.
func fakeStdoutTerminal(t *testing.T) {
	t.Helper()
	original := stdoutIsTerminal
	stdoutIsTerminal = func(*Meta) bool { return true }
	t.Cleanup(func() { stdoutIsTerminal = original })
}

func TestOutputJSON(t *testing.T) {
	c, stdout, _ := newExecuteCommand(nil)
	if err := c.OutputJSON(map[string]int{"count": 2}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "{\n  \"count\": 2\n}\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestOutputTable(t *testing.T) {
	headers := []string{"Name", "Flavor Type"}
	rows := [][]string{{"red", "cherry"}, {"purple", "grape\tjuice"}}

	c, stdout, _ := newExecuteCommand(nil)
	if err := c.OutputTable(headers, rows); err != nil {
		t.Fatal(err)
	}
	want := "NAME     FLAVOR TYPE\nred      cherry\npurple   grape juice\n"
	if stdout.String() != want {
		t.Errorf("table = %q, want %q", stdout.String(), want)
	}

	c, stdout, _ = newExecuteCommand(nil)
	c.Meta.flags.plain = true
	if err := c.OutputTable(headers, rows); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "red\tcherry\npurple\tgrape juice\n" {
		t.Errorf("plain = %q", stdout.String())
	}

	c, stdout, _ = newExecuteCommand(nil)
	c.Meta.flags.json = true
	if err := c.OutputTable(headers, [][]string{{"red"}}); err != nil {
		t.Fatal(err)
	}
	var records []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0]["name"] != "red" || records[0]["flavor_type"] != "" {
		t.Errorf("json = %v", records)
	}

	c, stdout, _ = newExecuteCommand(nil)
	c.Meta.flags.plain = true
	c.OutputTable(headers, nil)
	if stdout.Len() != 0 {
		t.Errorf("an empty plain table should print nothing: %q", stdout.String())
	}
}

func TestOutputTableBoldHeaderOnColor(t *testing.T) {
	c, stdout, _ := newExecuteCommand(nil)
	c.Ui.(*settingsUi).Ui.(*TerminalUi).OutputColor = true
	c.OutputTable([]string{"name"}, [][]string{{"red"}})
	if !strings.HasPrefix(stdout.String(), "\x1b[1mNAME\x1b[0m\n") {
		t.Errorf("table = %q", stdout.String())
	}
}

func TestSuccess(t *testing.T) {
	c, stdout, _ := newExecuteCommand(nil)
	c.Success("Ate 2 lollipops")
	if stdout.String() != "Ate 2 lollipops\n" {
		t.Errorf("stdout = %q", stdout.String())
	}

	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	c, stdout, _ = newExecuteCommand(nil)
	c.Ui.(*settingsUi).Ui.(*TerminalUi).OutputColor = true
	c.Success("Ate 2 lollipops")
	if stdout.String() != "\x1b[32m✓\x1b[0m Ate 2 lollipops\n" {
		t.Errorf("color stdout = %q", stdout.String())
	}

	c, stdout, _ = newExecuteCommand(nil)
	c.Meta.flags.json = true
	c.Success("Ate 2 lollipops")
	if stdout.Len() != 0 {
		t.Errorf("Success should be omitted under --json: %q", stdout.String())
	}
}

func TestSuggestNextCommands(t *testing.T) {
	c, stdout, stderr := newExecuteCommand(nil)
	c.Suggest("test-app status")
	if stdout.Len() != 0 || stderr.String() != "Next, run 'test-app status'.\n" {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	c, _, stderr = newExecuteCommand(nil)
	c.Suggest("test-app status", "test-app logs")
	if stderr.String() != "Next steps:\n  test-app status\n  test-app logs\n" {
		t.Errorf("stderr = %q", stderr.String())
	}

	c, _, stderr = newExecuteCommand(&Settings{Quiet: true})
	c.Suggest("test-app status")
	if stderr.Len() != 0 {
		t.Errorf("Suggest should be hidden under --quiet: %q", stderr.String())
	}

	c, _, stderr = newExecuteCommand(nil)
	c.Meta.flags.json = true
	c.Suggest("test-app status")
	if stderr.Len() != 0 {
		t.Errorf("Suggest should be hidden under --json: %q", stderr.String())
	}
}

func TestSupportsUTF8(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "C")
	os.Unsetenv("LC_ALL")
	os.Unsetenv("LC_CTYPE")
	if supportsUTF8() && successMark() != "✓" {
		t.Fatal("inconsistent mark")
	}

	t.Setenv("LC_ALL", "en_US.utf8")
	if !supportsUTF8() {
		t.Fatal("utf8 locale not detected")
	}
}

func TestPagerCommand(t *testing.T) {
	for _, name := range []string{"TEST_APP_PAGER", "PAGER"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	if got := PagerCommand("test-app"); got != "less" {
		t.Errorf("default = %q", got)
	}

	t.Setenv("PAGER", "more")
	if got := PagerCommand("test-app"); got != "more" {
		t.Errorf("PAGER = %q", got)
	}

	t.Setenv("TEST_APP_PAGER", "")
	if got := PagerCommand("test-app"); got != "" {
		t.Errorf("an empty app pager should disable paging: %q", got)
	}
}

func TestOutputPagedWithoutTerminal(t *testing.T) {
	c, stdout, _ := newExecuteCommand(nil)
	if err := c.OutputPaged("long output"); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "long output\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestOutputPagedRespectsNoPager(t *testing.T) {
	fakeTerminal(t)
	fakeStdoutTerminal(t)
	t.Setenv("PAGER", "false")

	c, stdout, _ := newExecuteCommand(&Settings{NoPager: true})
	c.OutputPaged("long output")
	if stdout.String() != "long output\n" {
		t.Errorf("--no-pager should write directly: %q", stdout.String())
	}
}

func TestIsInteractive(t *testing.T) {
	c, _ := newPromptCommand("", nil)
	if c.IsInteractive() {
		t.Fatal("buffers are not interactive")
	}

	fakeTerminal(t)
	fakeStdoutTerminal(t)
	if !c.IsInteractive() || !c.IsStdoutTerminal() {
		t.Fatal("fake terminals should be interactive")
	}
	if c.IsStderrTerminal() {
		t.Fatal("a buffer is not a terminal")
	}
}

func TestMainHelpIsNotPagedWhenShort(t *testing.T) {
	commands := map[string]*testCommand{"eat": {name: "eat", flags: FlagSetClient}}
	t.Setenv("LINES", "1000")

	got := runTestMain(t, []string{"eat", "--help"}, commands)
	if got.code != 0 || !strings.Contains(got.stdout, "USAGE") {
		t.Fatalf("code = %d, stdout = %q", got.code, got.stdout)
	}
}
