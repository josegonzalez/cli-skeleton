package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

// completionShells are the shells the completion command supports.
var completionShells = []string{"bash", "zsh", "fish"}

// completionCommand prints and installs shell completion scripts. The
// scripts call back into the cli tool, which answers with the predictions
// of its commands through posener/complete.
type completionCommand struct {
	Meta

	install   bool
	uninstall bool
}

func (c *completionCommand) Name() string { return "completion" }
func (c *completionCommand) Help() string { return CommandHelp(c) }

func (c *completionCommand) Synopsis() string {
	return "Print or install the shell completion script"
}

func (c *completionCommand) Description() string {
	return "Prints the completion script for bash, zsh or fish. With --install, the script is written to the completion directory of the shell, which the shell loads on its own, after asking for confirmation. Shell startup files such as ~/.bashrc are never changed."
}

func (c *completionCommand) Arguments() []Argument {
	return []Argument{{
		Name:        "shell",
		Description: "the shell to complete for: bash, zsh or fish; defaults to $SHELL",
		Optional:    true,
		Type:        ArgumentString,
		Validate: func(value string) error {
			for _, shell := range completionShells {
				if value == shell {
					return nil
				}
			}
			return fmt.Errorf("must be one of %s", strings.Join(completionShells, ", "))
		},
	}}
}

func (c *completionCommand) Examples() []Example {
	app := c.AppName()
	return []Example{
		{Description: "Install completion for the current shell", Command: app + " completion --install"},
		{Description: "Print the zsh completion script", Command: app + " completion zsh"},
		{Description: "Remove the installed bash completion", Command: app + " completion bash --uninstall"},
	}
}

func (c *completionCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), FlagSetClient|FlagSetForce)
	f.BoolVar(&c.install, "install", false, "write the completion script to the completion directory of the shell")
	f.BoolVar(&c.uninstall, "uninstall", false, "remove the installed completion script")
	return f
}

func (c *completionCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictSet(completionShells...)
}

func (c *completionCommand) AutocompleteFlags() complete.Flags {
	return MergeAutocompleteFlags(c.Meta.AutocompleteFlags(FlagSetClient|FlagSetForce), complete.Flags{
		"--install":   complete.PredictNothing,
		"--uninstall": complete.PredictNothing,
	})
}

func (c *completionCommand) Run(args []string) int {
	return Execute(c, args, func(arguments map[string]Argument) error {
		if c.install && c.uninstall {
			return NewUsageError("--install and --uninstall cannot be used together")
		}

		shell := arguments["shell"].StringValue()
		if shell == "" {
			shell = filepath.Base(os.Getenv("SHELL"))
		}
		if !isCompletionShell(shell) {
			return &UsageError{
				Err:  fmt.Errorf("unable to detect a supported shell from SHELL=%q", os.Getenv("SHELL")),
				Hint: "Pass the shell to complete for: " + strings.Join(completionShells, ", "),
			}
		}

		binary, err := os.Executable()
		if err != nil {
			return err
		}

		script := completionScript(shell, c.AppName(), binary)
		path, err := completionPath(shell, c.AppName())
		if err != nil {
			return err
		}

		switch {
		case c.uninstall:
			return c.removeScript(shell, path)
		case c.install:
			return c.installScript(shell, path, script)
		default:
			c.writeOut(script)
			return nil
		}
	})
}

// installScript writes the completion script after the user agrees.
func (c *completionCommand) installScript(shell string, path string, script string) error {
	ok, err := c.Confirm(fmt.Sprintf("Write the %s completion script to %s?", shell, path))
	if err != nil {
		return err
	}
	if !ok {
		c.Ui.Info("Nothing was changed.")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		return err
	}

	c.Success(fmt.Sprintf("Installed %s completion to %s", shell, path))
	switch shell {
	case "bash":
		c.Ui.Info("Completion loads in new shells when bash-completion is installed.")
	case "zsh":
		c.Ui.Info(fmt.Sprintf("If completion does not load in new shells, add this line to ~/.zshrc before compinit:\n  fpath=(%s $fpath)", filepath.Dir(path)))
	}
	return nil
}

// removeScript removes the installed completion script.
func (c *completionCommand) removeScript(shell string, path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		c.Ui.Info(fmt.Sprintf("No %s completion is installed at %s.", shell, path))
		return nil
	}
	if err != nil {
		return err
	}
	c.Success(fmt.Sprintf("Removed %s completion from %s", shell, path))
	return nil
}

func isCompletionShell(shell string) bool {
	for _, s := range completionShells {
		if s == shell {
			return true
		}
	}
	return false
}

// completionPath returns where the completion script of a shell is
// installed. Each shell loads scripts from these directories on its own.
func completionPath(shell string, appName string) (string, error) {
	data, err := xdgBase("XDG_DATA_HOME", filepath.Join(".local", "share"))
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(data, "bash-completion", "completions", appName), nil
	case "zsh":
		return filepath.Join(data, "zsh", "site-functions", "_"+appName), nil
	case "fish":
		config, err := xdgBase("XDG_CONFIG_HOME", ".config")
		if err != nil {
			return "", err
		}
		return filepath.Join(config, "fish", "completions", appName+".fish"), nil
	}
	return "", fmt.Errorf("unsupported shell %q", shell)
}

// xdgBase returns an XDG base directory, without the app name.
func xdgBase(env string, homeFallback string) (string, error) {
	if base := os.Getenv(env); base != "" && filepath.IsAbs(base) {
		return base, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, homeFallback), nil
}

// completionScript returns the completion script of a shell. The scripts
// set COMP_LINE and run the cli tool, which prints the completions.
func completionScript(shell string, appName string, binary string) string {
	quoted := shellQuote(binary)
	function := "__complete_" + strings.NewReplacer("-", "_", ".", "_").Replace(appName)

	switch shell {
	case "bash":
		return fmt.Sprintf("# bash completion for %[1]s\ncomplete -o default -C %[2]s %[1]s\n", appName, quoted)
	case "zsh":
		return fmt.Sprintf(`#compdef %[1]s
# zsh completion for %[1]s
local -a completions
completions=("${(@f)$(COMP_LINE="${(j: :)words[1,CURRENT]}" %[2]s 2>/dev/null)}")
compadd -- "${completions[@]}"
`, appName, quoted)
	case "fish":
		return fmt.Sprintf(`# fish completion for %[1]s
function %[3]s
    set -lx COMP_LINE (commandline -cp)
    test -z (commandline -ct)
    and set COMP_LINE "$COMP_LINE "
    %[2]s
end
complete -f -c %[1]s -a "(%[3]s)"
`, appName, quoted, function)
	}
	return ""
}

// shellQuote quotes s for POSIX shells and fish.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// rewriteAutocompleteArgs sends mitchellh/cli's autocomplete install flags
// to the completion command, which asks before writing anything and never
// edits shell startup files.
func rewriteAutocompleteArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch args[0] {
	case "-autocomplete-install", "--autocomplete-install":
		return append([]string{"completion", "--install"}, args[1:]...)
	case "-autocomplete-uninstall", "--autocomplete-uninstall":
		return append([]string{"completion", "--uninstall"}, args[1:]...)
	}
	return args
}

// isCompletionRequest reports whether the shell is asking for completions.
func isCompletionRequest() bool {
	return os.Getenv("COMP_LINE") != ""
}
