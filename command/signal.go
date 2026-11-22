package command

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// DefaultGracePeriod is how long a command has to stop after the first
// interrupt before the process exits.
const DefaultGracePeriod = 10 * time.Second

// interruptMessage is printed on the first interrupt.
const interruptMessage = "Gracefully stopping... (press Ctrl+C again to force)"

// WithGracePeriod sets how long a command has to stop after the first
// interrupt before the process exits with ExitCodeInterrupt. A zero or
// negative duration waits until the command returns or a second interrupt
// arrives.
func WithGracePeriod(d time.Duration) Option {
	return func(o *mainOptions) {
		o.gracePeriod = &d
	}
}

// interruptHook is run when the process is interrupted.
type interruptHook struct {
	fn func()

	// immediate makes the first interrupt exit right away, without the
	// graceful-stop message. Prompts use it, since there is nothing to
	// clean up while waiting for input.
	immediate bool
}

// interruptHandler cancels the run context on the first SIGINT or SIGTERM
// and exits the process on the second one, or when the grace period runs
// out.
type interruptHandler struct {
	mu          sync.Mutex
	cancel      context.CancelFunc
	warn        func(string)
	exit        func(int)
	gracePeriod time.Duration
	hooks       map[int]interruptHook
	nextID      int
	count       int
	signals     chan os.Signal
	done        chan struct{}
	stopOnce    sync.Once
}

func newInterruptHandler(cancel context.CancelFunc, warn func(string), gracePeriod time.Duration, exit func(int)) *interruptHandler {
	return &interruptHandler{
		cancel:      cancel,
		warn:        warn,
		exit:        exit,
		gracePeriod: gracePeriod,
		hooks:       map[int]interruptHook{},
		done:        make(chan struct{}),
	}
}

// start begins listening for SIGINT and SIGTERM.
func (h *interruptHandler) start() {
	h.signals = make(chan os.Signal, 2)
	signal.Notify(h.signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-h.signals:
				h.interrupt()
			case <-h.done:
				return
			}
		}
	}()
}

// stop stops listening for signals and restores the default behavior.
func (h *interruptHandler) stop() {
	h.stopOnce.Do(func() {
		if h.signals != nil {
			signal.Stop(h.signals)
		}
		close(h.done)
	})
}

// register adds a hook run on interrupt, and returns a function that removes
// it.
func (h *interruptHandler) register(hook interruptHook) func() {
	if h == nil {
		return func() {}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	h.hooks[id] = hook

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.hooks, id)
	}
}

// interrupt handles one SIGINT or SIGTERM.
func (h *interruptHandler) interrupt() {
	h.mu.Lock()
	h.count++
	count := h.count
	hooks := make([]interruptHook, 0, len(h.hooks))
	immediate := false
	for id := h.nextID - 1; id >= 0; id-- {
		if hook, ok := h.hooks[id]; ok {
			hooks = append(hooks, hook)
			immediate = immediate || hook.immediate
		}
	}
	h.mu.Unlock()

	if count > 1 {
		h.exit(ExitCodeInterrupt)
		return
	}

	for _, hook := range hooks {
		if hook.fn != nil {
			hook.fn()
		}
	}

	if immediate {
		h.exit(ExitCodeInterrupt)
		return
	}

	h.warn(interruptMessage)
	h.cancel()

	if h.gracePeriod > 0 {
		go func() {
			select {
			case <-time.After(h.gracePeriod):
				h.exit(ExitCodeInterrupt)
			case <-h.done:
			}
		}()
	}
}

// onInterrupt registers fn to run on the first interrupt, before the context
// is cancelled. It returns a function that removes the hook.
func (m *Meta) onInterrupt(fn func()) func() {
	return m.interrupts.register(interruptHook{fn: fn})
}

// onInterruptExit registers fn to run on the first interrupt, after which
// the process exits immediately with ExitCodeInterrupt. It returns a
// function that removes the hook.
func (m *Meta) onInterruptExit(fn func()) func() {
	return m.interrupts.register(interruptHook{fn: fn, immediate: true})
}
