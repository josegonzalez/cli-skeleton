package command

import (
	"context"
	"os"
	"strconv"
	"strings"
)

func SetupRun(ctx context.Context, appName string, version string, args []string) *Meta {
	// Create the meta object
	metaPtr := new(Meta)
	metaPtr.appName = appName
	metaPtr.colorDisabled = scanNoColor(args)
	metaPtr.Ui = NewTerminalUi(appName, metaPtr.colorDisabled)

	os.Setenv("CLI_APP_NAME", appName)
	os.Setenv("CLI_VERSION", version)
	metaPtr.Context = ctx
	return metaPtr
}

// scanNoColor reports whether --no-color was passed in args. The UI is
// created before any flags are parsed, so the flag is detected ahead of time.
// Scanning stops at `--`, which ends flag parsing.
func scanNoColor(args []string) bool {
	noColor := false
	for _, arg := range args {
		if arg == "--" {
			break
		}

		if arg == "--no-color" {
			noColor = true
			continue
		}

		if value, ok := strings.CutPrefix(arg, "--no-color="); ok {
			if b, err := strconv.ParseBool(value); err == nil {
				noColor = b
			}
		}
	}

	return noColor
}
