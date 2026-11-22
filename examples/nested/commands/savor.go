package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/josegonzalez/cli-skeleton/command"
	"github.com/posener/complete"
	flag "github.com/spf13/pflag"
)

const savorFlags = command.FlagSetClient | command.FlagSetTimeout

type SavorCommand struct {
	command.Meta

	duration time.Duration
	wait     bool
	fail     bool
}

func (c *SavorCommand) Name() string {
	return "lollipop savor"
}

func (c *SavorCommand) Synopsis() string {
	return "Slowly savor a lollipop"
}

func (c *SavorCommand) Description() string {
	return "Savoring takes a while, so progress is shown as it happens. Only one lollipop can be savored at a time; pass --wait to wait for another one to finish. Press Ctrl+C to stop."
}

func (c *SavorCommand) Help() string {
	return command.CommandHelp(c)
}

func (c *SavorCommand) Examples() []command.Example {
	appName := c.AppName()
	return []command.Example{
		{Description: "Savor a lollipop", Command: appName + " " + c.Name()},
		{Description: "Savor a lollipop for ten seconds", Command: appName + " " + c.Name() + " --duration 10s"},
		{Description: "Wait for another lollipop to be finished first", Command: appName + " " + c.Name() + " --wait"},
	}
}

func (c *SavorCommand) Arguments() []command.Argument {
	return []command.Argument{}
}

func (c *SavorCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *SavorCommand) FlagSet() *flag.FlagSet {
	f := c.Meta.FlagSet(c.Name(), savorFlags)
	f.DurationVar(&c.duration, "duration", 3*time.Second, "how long to savor the lollipop")
	f.BoolVar(&c.wait, "wait", false, "wait for another lollipop to be finished instead of failing")
	f.BoolVar(&c.fail, "drop", false, "drop the lollipop halfway through, to show a failed step")
	return f
}

func (c *SavorCommand) AutocompleteFlags() complete.Flags {
	return command.MergeAutocompleteFlags(
		c.Meta.AutocompleteFlags(savorFlags),
		complete.Flags{
			"--duration": complete.PredictAnything,
			"--wait":     complete.PredictNothing,
			"--drop":     complete.PredictNothing,
		},
	)
}

func (c *SavorCommand) Run(args []string) int {
	return command.Execute(c, args, func(map[string]command.Argument) error {
		ctx, cancel := c.TimeoutContext()
		defer cancel()

		var lock *command.Lock
		var err error
		if c.wait {
			lock, err = c.LockWait(ctx, "savor")
		} else {
			lock, err = c.Lock("savor")
		}
		if err != nil {
			return err
		}
		defer lock.Unlock()

		steps := c.Steps(2)
		err = steps.Run("Unwrapping the lollipop", func(step *command.Step) error {
			step.Printf("peeling the wrapper")
			step.Printf("folding the wrapper")
			step.Result("1 wrapper")
			return nil
		})
		if err != nil {
			return err
		}

		err = steps.Run("Checking the recipe", func(step *command.Step) error {
			self, err := os.Executable()
			if err != nil {
				return err
			}
			return step.Exec(exec.Command(self, "version", "--json"))
		})
		if err != nil {
			return err
		}

		if err := c.savor(ctx); err != nil {
			return err
		}

		steps.Summary()
		c.Success("Savored a lollipop")
		return nil
	})
}

// savor waits for the duration with a progress bar, until ctx is done.
func (c *SavorCommand) savor(ctx context.Context) error {
	const tick = 100 * time.Millisecond
	total := int64(c.duration / tick)
	if total < 1 {
		total = 1
	}

	bar := c.ProgressBar("Savoring", total)
	for i := int64(1); i <= total; i++ {
		select {
		case <-ctx.Done():
			return bar.Done(ctx.Err())
		case <-time.After(tick):
		}

		bar.Add(1)
		if i%10 == 0 {
			bar.Printf("lick %d of %d", i/10, total/10)
		}
		if c.fail && i == total/2 {
			bar.Printf("the lollipop slipped")
			return bar.Done(errors.New("dropped the lollipop"))
		}
	}

	bar.Result(fmt.Sprintf("savored for %s", c.duration))
	return bar.Done(nil)
}
