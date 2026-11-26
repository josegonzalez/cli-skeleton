package commands

import (
	"fmt"
	"strings"

	"github.com/josegonzalez/cli-skeleton/internal/scaffold"
)

// Asker asks questions and reports problems with the answers. cli.Ui
// implements it.
type Asker interface {
	Ask(query string) (string, error)
	Error(message string)
}

// prompter builds a spec from answers to questions.
type prompter struct {
	ui Asker
}

// PromptSpec asks for the app details and its commands, and returns the
// resulting spec. defaultName is offered as the app name.
func PromptSpec(ui Asker, defaultName string) (*scaffold.Spec, error) {
	p := &prompter{ui: ui}
	spec := &scaffold.Spec{}

	var err error
	for {
		if spec.Name, err = p.ask("App name", defaultName); err != nil {
			return nil, err
		}
		if p.check(&scaffold.Spec{Name: spec.Name, Commands: []scaffold.Command{placeholderCommand}}) {
			break
		}
	}
	for {
		module, err := p.ask("Go module path", spec.Name)
		if err != nil {
			return nil, err
		}
		if module != spec.Name {
			spec.Module = module
		}
		if p.check(&scaffold.Spec{Name: spec.Name, Module: spec.Module, Commands: []scaffold.Command{placeholderCommand}}) {
			break
		}
	}
	if spec.Description, err = p.ask("Description (optional)", ""); err != nil {
		return nil, err
	}
	for {
		ui, err := p.ask("Log output: "+strings.Join(scaffold.UIs, ", "), scaffold.UIDefault)
		if err != nil {
			return nil, err
		}
		if contains(scaffold.UIs, ui) {
			if ui != scaffold.UIDefault {
				spec.UI = ui
			}
			break
		}
		p.ui.Error(fmt.Sprintf("Pick one of %s.", strings.Join(scaffold.UIs, ", ")))
	}
	if spec.Config, err = p.confirm("Read flag defaults from config files?"); err != nil {
		return nil, err
	}

	for {
		c, done, err := p.command(spec)
		if err != nil {
			return nil, err
		}
		if done {
			if len(spec.Commands) > 0 {
				break
			}
			p.ui.Error("Add at least one command.")
			continue
		}
		spec.Commands = append(spec.Commands, c)
	}

	return spec, nil
}

// placeholderCommand stands in for the commands that are not entered yet,
// so a partial spec can be validated.
var placeholderCommand = scaffold.Command{Name: "placeholder", Synopsis: "placeholder"}

// command asks for one command. done is true when no name was entered.
func (p *prompter) command(spec *scaffold.Spec) (c scaffold.Command, done bool, err error) {
	for {
		if c.Name, err = p.ask(`Command name, such as "deploy" or "app create" (leave empty to finish)`, ""); err != nil {
			return c, false, err
		}
		if c.Name == "" {
			return c, true, nil
		}
		c.Synopsis = "placeholder"
		if p.checkCommand(spec, c) {
			break
		}
	}

	if c.Synopsis, err = p.required("Synopsis, shown in the command list"); err != nil {
		return c, false, err
	}
	if c.Description, err = p.ask("Description (optional)", ""); err != nil {
		return c, false, err
	}

	for {
		sets, err := p.ask("Standard flags, comma-separated: "+strings.Join(scaffold.StandardFlagSets, ", ")+" (optional)", "")
		if err != nil {
			return c, false, err
		}
		c.FlagSets = nil
		for _, name := range strings.Split(sets, ",") {
			if name = strings.TrimSpace(name); name != "" {
				c.FlagSets = append(c.FlagSets, name)
			}
		}
		if len(c.FlagSets) > 0 {
			c.FlagSets = append([]string{"client"}, c.FlagSets...)
		}
		if p.checkCommand(spec, c) {
			break
		}
	}

	for {
		name, err := p.ask("Argument name (leave empty to finish)", "")
		if err != nil {
			return c, false, err
		}
		if name == "" {
			break
		}
		a := scaffold.Argument{Name: name}
		if a.Description, err = p.required("Argument description"); err != nil {
			return c, false, err
		}
		if a.Type, err = p.ask("Argument type: string, int, bool, float, duration, file or list", "string"); err != nil {
			return c, false, err
		}
		if a.Type == "string" {
			a.Type = ""
		}
		if a.Optional, err = p.confirm("Is the argument optional?"); err != nil {
			return c, false, err
		}

		candidate := c
		candidate.Arguments = append(append([]scaffold.Argument{}, c.Arguments...), a)
		if p.checkCommand(spec, candidate) {
			c = candidate
		}
	}

	for {
		name, err := p.ask("Flag name, without dashes (leave empty to finish)", "")
		if err != nil {
			return c, false, err
		}
		if name == "" {
			break
		}
		f := scaffold.Flag{Name: name}
		if f.Type, err = p.ask("Flag type: "+strings.Join(scaffold.FlagTypes, ", "), "string"); err != nil {
			return c, false, err
		}
		if f.Type == "string" {
			f.Type = ""
		}
		if f.Default, err = p.ask("Default value (optional)", ""); err != nil {
			return c, false, err
		}
		if f.Usage, err = p.required("Flag description"); err != nil {
			return c, false, err
		}

		candidate := c
		candidate.Flags = append(append([]scaffold.Flag{}, c.Flags...), f)
		if p.checkCommand(spec, candidate) {
			c = candidate
		}
	}

	for {
		description, err := p.ask("Example description (leave empty to finish)", "")
		if err != nil {
			return c, false, err
		}
		if description == "" {
			break
		}
		e := scaffold.Example{Description: description}
		if e.Args, err = p.ask(fmt.Sprintf("Example arguments and flags, after %q (optional)", spec.Name+" "+c.Name), ""); err != nil {
			return c, false, err
		}

		candidate := c
		candidate.Examples = append(append([]scaffold.Example{}, c.Examples...), e)
		if p.checkCommand(spec, candidate) {
			c = candidate
		}
	}

	return c, false, nil
}

// ask asks a question, returning fallback for an empty answer.
func (p *prompter) ask(question string, fallback string) (string, error) {
	query := question + ":"
	if fallback != "" {
		query = fmt.Sprintf("%s [%s]:", question, fallback)
	}
	answer, err := p.ui.Ask(query)
	if err != nil {
		return "", err
	}
	if answer = strings.TrimSpace(answer); answer == "" {
		return fallback, nil
	}
	return answer, nil
}

// required asks a question until it is answered.
func (p *prompter) required(question string) (string, error) {
	for {
		answer, err := p.ask(question, "")
		if err != nil || answer != "" {
			return answer, err
		}
		p.ui.Error("An answer is required.")
	}
}

// confirm asks a yes or no question, defaulting to no.
func (p *prompter) confirm(question string) (bool, error) {
	for {
		answer, err := p.ask(question+" [y/N]", "")
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		}
		p.ui.Error("Answer y or n.")
	}
}

// check reports the problems with spec, returning true when there are
// none.
func (p *prompter) check(spec *scaffold.Spec) bool {
	err := spec.Validate()
	if err == nil {
		return true
	}
	if verr, ok := err.(*scaffold.ValidationError); ok {
		for _, problem := range verr.Problems {
			p.ui.Error(problem)
		}
		return false
	}
	p.ui.Error(err.Error())
	return false
}

// checkCommand validates spec with c added to it.
func (p *prompter) checkCommand(spec *scaffold.Spec, c scaffold.Command) bool {
	candidate := *spec
	candidate.Commands = append(append([]scaffold.Command{}, spec.Commands...), c)
	return p.check(&candidate)
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
