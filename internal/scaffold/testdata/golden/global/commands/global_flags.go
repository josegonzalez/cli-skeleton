package commands

import (
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

// GlobalFlagCommand holds the app-global flags, which every command accepts
// before or after its name.
type GlobalFlagCommand struct {
	global       bool
	globalString string
}

func (c *GlobalFlagCommand) GlobalFlags(f *flag.FlagSet) {
	f.BoolVar(&c.global, "global", false, "a bool global flag")
	f.StringVar(&c.globalString, "global-string", "", "a string global flag")
}

func (c *GlobalFlagCommand) AutocompleteGlobalFlags() complete.Flags {
	return complete.Flags{
		"--global":        complete.PredictNothing,
		"--global-string": complete.PredictAnything,
	}
}
