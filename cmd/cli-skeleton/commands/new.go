package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/josegonzalez/cli-skeleton/internal/scaffold"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const newFlags = command.FlagSetClient | command.FlagSetForce | command.FlagSetDryRun

type NewCommand struct {
	command.Meta

	spec         string
	saveSpec     string
	noTidy       bool
	skeletonPath string
}

func (c *NewCommand) Name() string {
	return "new"
}

func (c *NewCommand) Synopsis() string {
	return "Set up a new cli tool"
}

func (c *NewCommand) Description() string {
	return "Asks for the details of the app and its commands, then writes a project with a main.go, a test that checks the commands and one file per command, ready to fill in. Pass --spec to read the answers from a YAML file instead. Existing files are never overwritten unless --force is passed."
}

func (c *NewCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *NewCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Answer questions to set up an app in the current directory", Command: appName + " " + c.Name()},
		{Description: "Set up an app in a new directory from a spec", Command: appName + " " + c.Name() + " my-app --spec my-app.yaml"},
		{Description: "Save the answers as a spec for later", Command: appName + " " + c.Name() + " my-app --save-spec my-app.yaml"},
		{Description: "See which files would be written", Command: appName + " " + c.Name() + " my-app --spec my-app.yaml --dry-run"},
	}
}

func (c *NewCommand) Arguments() []command.Argument {
	return []command.Argument{
		{
			Name:        "directory",
			Description: "the directory to set up the app in, created when missing; defaults to the current directory",
			Optional:    true,
			Type:        command.ArgumentString,
		},
	}
}

func (c *NewCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("*")
}

func (c *NewCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), newFlags)
	f.StringVar(&c.spec, "spec", "", "read the app details from a YAML spec instead of asking; - reads stdin")
	f.StringVar(&c.saveSpec, "save-spec", "", "save the answers as a YAML spec; - writes to stdout")
	f.BoolVar(&c.noTidy, "no-tidy", false, "skip running go get and go mod tidy")
	f.StringVar(&c.skeletonPath, "skeleton-path", "", "use the cli-skeleton checkout at this path instead of a released version")
	return f
}

func (c *NewCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(newFlags),
		complete.Flags{
			"--spec":          complete.PredictFiles("*.yaml"),
			"--save-spec":     complete.PredictFiles("*.yaml"),
			"--no-tidy":       complete.PredictNothing,
			"--skeleton-path": complete.PredictDirs("*"),
		},
	)
}

func (c *NewCommand) Run(args []string) int {
	return command.Execute(c, args, func(arguments map[string]command.Argument) error {
		dir := arguments["directory"].StringValue()
		if dir == "" {
			dir = "."
		}
		return c.run(dir)
	})
}

func (c *NewCommand) run(dir string) error {
	if c.spec != "" && c.saveSpec != "" {
		return command.NewUsageError("--spec and --save-spec cannot be used together")
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	spec, err := c.loadSpec(filepath.Base(absDir))
	if err != nil {
		return err
	}

	if c.saveSpec != "" {
		if err := c.writeSpec(spec); err != nil {
			return err
		}
	}

	opts := scaffold.Options{}
	if c.skeletonPath != "" {
		skeletonPath, err := filepath.Abs(c.skeletonPath)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(skeletonPath, "go.mod")); err != nil {
			return command.NewUserError(fmt.Sprintf("%s is not a cli-skeleton checkout", c.skeletonPath), "Pass the directory that holds the go.mod of cli-skeleton to --skeleton-path.")
		}
		if rel, err := filepath.Rel(absDir, skeletonPath); err == nil {
			skeletonPath = rel
		}
		opts.SkeletonPath = filepath.ToSlash(skeletonPath)
	}

	files, err := scaffold.Generate(spec, opts)
	if err != nil {
		return command.NewUsageError("%v", err)
	}

	goCommands := c.goCommands()

	if c.IsDryRun() {
		for _, file := range files {
			c.Ui.Output(fmt.Sprintf("Would write %s", filepath.Join(dir, filepath.FromSlash(file.Path))))
		}
		for _, args := range goCommands {
			c.Ui.Output(fmt.Sprintf("Would run '%s' in %s", strings.Join(args, " "), dir))
		}
		return nil
	}

	if !c.IsForce() {
		var existing []string
		for _, file := range files {
			path := filepath.Join(dir, filepath.FromSlash(file.Path))
			if _, err := os.Lstat(path); err == nil {
				existing = append(existing, path)
			}
		}
		if len(existing) > 0 {
			return command.NewUserError(
				"These files already exist:\n  "+strings.Join(existing, "\n  "),
				"Pass --force to overwrite them, or pick another directory.",
			)
		}
	}

	for _, file := range files {
		path := filepath.Join(dir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.Contents, 0o644); err != nil {
			return err
		}
	}

	if len(goCommands) > 0 {
		steps := c.Steps(len(goCommands))
		for _, args := range goCommands {
			args := args
			err := steps.Run("Running "+strings.Join(args, " "), func(step *command.Step) error {
				cmd := exec.Command(args[0], args[1:]...)
				cmd.Dir = dir
				return step.Exec(cmd)
			})
			if err != nil {
				return err
			}
		}
		steps.Summary()
	}

	noun := "commands"
	if len(spec.Commands) == 1 {
		noun = "command"
	}
	c.Success(fmt.Sprintf("Created %s with %d %s in %s", spec.Name, len(spec.Commands), noun, dir))

	next := []string{}
	if c.noTidy {
		for _, args := range c.tidyCommands() {
			next = append(next, strings.Join(args, " "))
		}
	}
	next = append(next, "go run . --help")
	if dir != "." {
		next[0] = fmt.Sprintf("cd %s && %s", dir, next[0])
	}
	c.Suggest(next...)
	return nil
}

// loadSpec reads the spec from --spec, or asks for it.
func (c *NewCommand) loadSpec(defaultName string) (*scaffold.Spec, error) {
	if c.spec == "" {
		if !c.CanPrompt() {
			return nil, &command.UsageError{
				Err:  errors.New("the app details are needed, but they cannot be asked for"),
				Hint: "Pass --spec with a YAML spec, or run this in a terminal without --no-input to answer questions.",
			}
		}
		return PromptSpec(c.Ui, defaultName)
	}

	r, err := command.OpenInput(c.spec)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	spec, err := scaffold.ParseSpec(r)
	if err != nil {
		return nil, command.NewUsageError("reading %s: %v", c.spec, err)
	}
	if spec.Name == "" {
		spec.Name = defaultName
	}
	if err := spec.Validate(); err != nil {
		return nil, command.NewUsageError("%s: %v", c.spec, err)
	}
	return spec, nil
}

// writeSpec saves the spec to --save-spec.
func (c *NewCommand) writeSpec(spec *scaffold.Spec) error {
	contents, err := spec.Marshal()
	if err != nil {
		return err
	}

	w, err := command.OpenOutput(c.saveSpec)
	if err != nil {
		return err
	}
	if _, err := w.Write(contents); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if c.saveSpec != "-" {
		c.Ui.Info(fmt.Sprintf("Saved the spec to %s", c.saveSpec))
	}
	return nil
}

// goCommands returns the go commands run after the files are written.
func (c *NewCommand) goCommands() [][]string {
	if c.noTidy {
		return nil
	}
	return c.tidyCommands()
}

// tidyCommands returns the go commands that fetch the dependencies of the
// generated app.
func (c *NewCommand) tidyCommands() [][]string {
	var commands [][]string
	if c.skeletonPath == "" {
		commands = append(commands, []string{"go", "get", scaffold.SkeletonModule + "@" + skeletonVersion()})
	}
	return append(commands, []string{"go", "mod", "tidy"})
}

// releasePattern matches tagged releases, but not pseudo-versions or
// builds with local changes.
var (
	releasePattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
	pseudoPattern  = regexp.MustCompile(`[0-9]{14}-[0-9a-f]{12}$`)
)

// skeletonVersion returns the version of cli-skeleton this binary was
// built from when it is a tagged release, such as with
// `go install github.com/josegonzalez/cli-skeleton/cmd/cli-skeleton@v1.0.0`,
// and latest otherwise.
func skeletonVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "latest"
	}
	return releaseOrLatest(info.Main.Version)
}

func releaseOrLatest(version string) string {
	if releasePattern.MatchString(version) && !pseudoPattern.MatchString(version) {
		return version
	}
	return "latest"
}
