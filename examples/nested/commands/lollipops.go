package commands

import (
	"fmt"
	"strings"
)

// Lollipop is a lollipop in the jar.
type Lollipop struct {
	Flavor string
	Color  string
	Eaten  bool
}

// lollipops is the jar of lollipops the example commands work with.
var lollipops = []Lollipop{
	{Flavor: "cherry", Color: "red"},
	{Flavor: "grape", Color: "purple"},
	{Flavor: "lemon", Color: "yellow"},
	{Flavor: "lime", Color: "green", Eaten: true},
}

// flavors returns the flavors in the jar.
func flavors() []string {
	names := make([]string, 0, len(lollipops))
	for _, l := range lollipops {
		names = append(names, l.Flavor)
	}
	return names
}

// validateFlavor checks that a flavor is in the jar.
func validateFlavor(flavor string) error {
	for _, l := range lollipops {
		if l.Flavor == flavor {
			return nil
		}
	}
	return fmt.Errorf("must be one of %s", strings.Join(flavors(), ", "))
}
