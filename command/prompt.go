package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// cancelHint is appended to free-text prompts so the way out is clear.
const cancelHint = "(Ctrl+C to cancel)"

// isTerminalReader reports whether r is an interactive terminal. Tests
// replace it to simulate a terminal.
var isTerminalReader = func(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && IsTerminal(f)
}

// stdinReader returns the stdin of the run.
func (m *Meta) stdinReader() io.Reader {
	if m.stdin == nil {
		return os.Stdin
	}
	return m.stdin
}

// CanPrompt reports whether the user can be prompted: stdin is a terminal
// and --no-input was not passed.
func (m *Meta) CanPrompt() bool {
	return checkPrompt(m.settings, m.stdinReader()) == nil
}

// Confirm asks a yes or no question, defaulting to no. It returns true
// without asking when -f/--force was passed. When the user cannot be
// prompted, it returns a UserError explaining how to confirm without a
// prompt.
func (m *Meta) Confirm(question string) (bool, error) {
	if m.flags.force {
		return true, nil
	}

	answer, err := m.Ui.Ask(question + " [y/N]")
	if err != nil {
		var noInput *NoInputError
		if errors.As(err, &noInput) {
			hint := "Run the command again from a terminal to confirm."
			if m.flagBits&FlagSetForce != 0 {
				hint = "Pass --force to continue without confirming."
			}
			return false, &UserError{Message: "Confirmation required, but " + noInput.Reason + ".", Hint: hint, Code: ExitCodeUsage}
		}
		return false, err
	}

	return isYes(answer), nil
}

// ConfirmName guards a severe action by asking the user to type the name of
// the resource it affects. Passing --confirm=<name> confirms without a
// prompt, so scripts can still run the command. It returns an error when
// the confirmation does not match or cannot be asked for.
func (m *Meta) ConfirmName(name string) error {
	if given := m.flags.confirm; given != "" {
		if given != name {
			return &UsageError{Err: fmt.Errorf("--confirm %q does not match %q", given, name)}
		}
		return nil
	}

	answer, err := m.Ui.Ask(fmt.Sprintf("This cannot be undone. Type %q to confirm %s:", name, cancelHint))
	if err != nil {
		var noInput *NoInputError
		if errors.As(err, &noInput) {
			hint := "Run the command again from a terminal to confirm."
			if m.flagBits&FlagSetConfirm != 0 {
				hint = fmt.Sprintf("Pass --confirm=%s to continue without a prompt.", name)
			}
			return &UserError{Message: "Confirmation required, but " + noInput.Reason + ".", Hint: hint, Code: ExitCodeUsage}
		}
		return err
	}

	if strings.TrimSpace(answer) != name {
		return &UserError{Message: fmt.Sprintf("Confirmation did not match %q, so nothing was changed.", name)}
	}
	return nil
}

// promptMissingArguments asks for missing required arguments that define a
// Prompt, when the user can be prompted.
func (m *Meta) promptMissingArguments(definitions []Argument, args []string) ([]string, error) {
	for i := len(args); i < len(definitions); i++ {
		definition := definitions[i]
		if definition.Optional || definition.Prompt == "" || definition.Type == ArgumentList {
			break
		}
		if !m.CanPrompt() {
			break
		}

		answer, err := m.Ui.Ask(definition.Prompt + " " + cancelHint)
		if err != nil {
			if errors.Is(err, ErrNoInput) {
				break
			}
			return nil, err
		}

		answer = strings.TrimSpace(answer)
		if answer == "" {
			return nil, &UsageError{Err: fmt.Errorf("%s is required", definition.Name)}
		}
		args = append(args, answer)
	}
	return args, nil
}

// RequireStdin returns a usage error when stdin is a terminal, for commands
// that read piped input. The error shows the concise command help instead
// of waiting for input that will never come.
func (m *Meta) RequireStdin() error {
	if isTerminalReader(m.stdinReader()) {
		return &UsageError{Err: errors.New("this command reads from stdin, but nothing was piped into it"), showUsage: true}
	}
	return nil
}

// Editor opens text in the user's editor and returns the edited text. The
// editor comes from VISUAL, then EDITOR, falling back to vi (notepad on
// Windows). It requires an interactive terminal.
func (m *Meta) Editor(initial string) (string, error) {
	if err := checkPrompt(m.settings, m.stdinReader()); err != nil {
		var noInput *NoInputError
		errors.As(err, &noInput)
		return "", &UserError{Message: "Unable to open an editor, because " + noInput.Reason + ".", Code: ExitCodeUsage}
	}

	f, err := m.TempFile("*.txt")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)

	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	editor := strings.Fields(EditorCommand())
	cmd := exec.CommandContext(m.context(), editor[0], append(editor[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor %q: %w", editor[0], err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EditorCommand returns the user's editor: VISUAL, then EDITOR, falling
// back to vi (notepad on Windows).
func EditorCommand() string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if editor := strings.TrimSpace(os.Getenv(name)); editor != "" {
			return editor
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// Shell returns the user's interactive shell: SHELL, falling back to
// /bin/sh (COMSPEC or cmd.exe on Windows).
func (m *Meta) Shell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			return comspec
		}
		return "cmd.exe"
	}
	return "/bin/sh"
}

// RunShell starts the user's interactive shell attached to the terminal,
// with env added to the environment, and waits for it to exit.
func (m *Meta) RunShell(ctx context.Context, env []string) error {
	if err := checkPrompt(m.settings, m.stdinReader()); err != nil {
		var noInput *NoInputError
		errors.As(err, &noInput)
		return &UserError{Message: "Unable to start an interactive shell, because " + noInput.Reason + ".", Code: ExitCodeUsage}
	}

	cmd := exec.CommandContext(ctx, m.Shell())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	return cmd.Run()
}

// context returns the run context, or a background context.
func (m *Meta) context() context.Context {
	if m.Context == nil {
		return context.Background()
	}
	return m.Context
}
