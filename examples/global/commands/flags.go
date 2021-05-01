package commands

import (
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

type GlobalFlagCommand struct {
	global      bool
	globalValue string
}

func (c *GlobalFlagCommand) GlobalFlags(f *flag.FlagSet) {
	f.BoolVar(&c.global, "global", false, "a bool global flag")
	f.StringVar(&c.globalValue, "global-string", "", "a string global flag")
}

func (c *GlobalFlagCommand) AutocompleteGlobalFlags() complete.Flags {
	return complete.Flags{
		"--global":        complete.PredictNothing,
		"--global-string": complete.PredictAnything,
	}
}
