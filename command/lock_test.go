package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	m := &Meta{appName: "test-app"}
	lock, err := m.Lock("deploy/production")
	if err != nil {
		t.Fatal(err)
	}

	if want := filepath.Join(dir, "test-app", "deploy_production.lock"); lock.Path() != want {
		t.Errorf("lock path = %q, want %q", lock.Path(), want)
	}

	_, err = m.Lock("deploy/production")
	var userErr *UserError
	if !errors.As(err, &userErr) {
		t.Fatalf("a held lock should be a UserError: %v", err)
	}
	if !strings.Contains(userErr.Message, "Another test-app process") || !strings.Contains(userErr.Message, "pid "+strconv.Itoa(os.Getpid())) {
		t.Errorf("message = %q", userErr.Message)
	}

	other, err := m.Lock("other")
	if err != nil {
		t.Fatalf("different locks should not conflict: %v", err)
	}
	other.Unlock()

	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("unlocking twice should be a no-op: %v", err)
	}

	again, err := m.Lock("deploy/production")
	if err != nil {
		t.Fatalf("a released lock should be available: %v", err)
	}
	again.Unlock()
}

func TestLockFallsBackToCacheDir(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)

	m := &Meta{appName: "test-app"}
	lock, err := m.Lock("deploy")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()

	if !strings.HasPrefix(lock.Path(), cache) {
		t.Fatalf("lock path = %q, want it under %q", lock.Path(), cache)
	}
}

func TestLockWait(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	m := &Meta{appName: "test-app"}

	held, err := m.Lock("deploy")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		held.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock, err := m.LockWait(ctx, "deploy")
	if err != nil {
		t.Fatalf("LockWait should succeed once the lock is released: %v", err)
	}
	lock.Unlock()
}

func TestLockWaitCancelled(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	m := &Meta{appName: "test-app"}

	held, err := m.Lock("deploy")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if _, err := m.LockWait(ctx, "deploy"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}
