package command

import (
	"os"

	colorable "github.com/mattn/go-colorable"
	"github.com/mitchellh/cli"
	"golang.org/x/crypto/ssh/terminal"
)

func SetupRun(appName string, version string, args []string) *Meta {
	// Parse flags into env vars for global use
	args = SetupEnv(args)

	// Create the meta object
	metaPtr := new(Meta)

	// Don't use color if disabled
	color := true
	if os.Getenv(EnvCLINoColor) != "" {
		color = false
	}

	isTerminal := terminal.IsTerminal(int(os.Stdout.Fd()))
	// Only use colored UI if stdout is a tty, and not disabled
	if isTerminal && color {
		metaPtr.Ui = &cli.ConcurrentUi{
			Ui: &cli.ColoredUi{
				ErrorColor: cli.UiColorRed,
				WarnColor:  cli.UiColorYellow,
				InfoColor:  cli.UiColorGreen,
				Ui: &cli.BasicUi{
					Reader:      os.Stdin,
					Writer:      colorable.NewColorableStdout(),
					ErrorWriter: colorable.NewColorableStderr(),
				},
			},
		}
	} else {
		metaPtr.Ui = &cli.ConcurrentUi{
			Ui: &cli.BasicUi{
				Reader:      os.Stdin,
				Writer:      colorable.NewColorableStdout(),
				ErrorWriter: colorable.NewColorableStderr(),
			},
		}
	}

	os.Setenv("CLI_APP_NAME", appName)
	os.Setenv("CLI_VERSION", version)

	return metaPtr
}

// setupEnv parses args and may replace them and sets some env vars to known
// values based on format options
func SetupEnv(args []string) []string {
	noColor := false
	for _, arg := range args {
		// Check if color is set
		if arg == "-no-color" || arg == "--no-color" {
			noColor = true
		}
	}

	// Put back into the env for later
	if noColor {
		os.Setenv(EnvCLINoColor, "true")
	}

	return args
}
