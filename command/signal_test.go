package command

import (
	"context"
	"sync"
	"testing"
	"time"
)

type exitRecorder struct {
	mu    sync.Mutex
	codes []int
	ch    chan int
}

func newExitRecorder() *exitRecorder {
	return &exitRecorder{ch: make(chan int, 10)}
}

func (r *exitRecorder) exit(code int) {
	r.mu.Lock()
	r.codes = append(r.codes, code)
	r.mu.Unlock()
	r.ch <- code
}

func (r *exitRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.codes)
}

type warnRecorder struct {
	mu       sync.Mutex
	messages []string
}

func (w *warnRecorder) warn(message string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages = append(w.messages, message)
}

func (w *warnRecorder) all() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.messages...)
}

func TestInterruptCancelsThenForces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exits := newExitRecorder()
	warns := &warnRecorder{}

	h := newInterruptHandler(cancel, warns.warn, 0, exits.exit)
	defer h.stop()

	var hookRan bool
	h.register(interruptHook{fn: func() { hookRan = true }})

	h.interrupt()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("first interrupt should cancel the context")
	}
	if !hookRan {
		t.Fatal("first interrupt should run hooks")
	}
	if got := warns.all(); len(got) != 1 || got[0] != interruptMessage {
		t.Fatalf("warnings = %v", got)
	}
	if exits.count() != 0 {
		t.Fatal("first interrupt should not exit")
	}

	h.interrupt()
	if code := <-exits.ch; code != ExitCodeInterrupt {
		t.Fatalf("exit code = %d", code)
	}
}

func TestInterruptGracePeriod(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	exits := newExitRecorder()

	h := newInterruptHandler(cancel, func(string) {}, 10*time.Millisecond, exits.exit)
	defer h.stop()

	h.interrupt()
	select {
	case code := <-exits.ch:
		if code != ExitCodeInterrupt {
			t.Fatalf("exit code = %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("grace period did not force an exit")
	}
}

func TestInterruptGracePeriodStopsWithHandler(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	exits := newExitRecorder()

	h := newInterruptHandler(cancel, func(string) {}, 50*time.Millisecond, exits.exit)
	h.interrupt()
	h.stop()

	time.Sleep(100 * time.Millisecond)
	if exits.count() != 0 {
		t.Fatal("grace period should not fire after the run finished")
	}
}

func TestInterruptImmediateHook(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exits := newExitRecorder()
	warns := &warnRecorder{}

	h := newInterruptHandler(cancel, warns.warn, 0, exits.exit)
	defer h.stop()

	var restored bool
	remove := h.register(interruptHook{fn: func() { restored = true }, immediate: true})

	h.interrupt()
	if code := <-exits.ch; code != ExitCodeInterrupt {
		t.Fatalf("exit code = %d", code)
	}
	if !restored {
		t.Fatal("immediate hook did not run")
	}
	if len(warns.all()) != 0 {
		t.Fatal("immediate exits should not print the graceful-stop message")
	}
	if ctx.Err() != nil {
		t.Fatal("immediate exits should not cancel the context")
	}
	remove()
}

func TestInterruptHookRemoval(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newInterruptHandler(cancel, func(string) {}, 0, func(int) {})
	defer h.stop()

	var ran bool
	remove := h.register(interruptHook{fn: func() { ran = true }})
	remove()

	h.interrupt()
	if ran {
		t.Fatal("removed hook should not run")
	}
}

func TestNilInterruptHandlerRegister(t *testing.T) {
	m := &Meta{}
	m.onInterrupt(func() {})()
	m.onInterruptExit(func() {})()
}
