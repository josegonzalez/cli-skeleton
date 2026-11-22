package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// lockPollInterval is how often LockWait retries a held lock.
const lockPollInterval = 100 * time.Millisecond

// errLockHeld is returned by tryLock when another process holds the lock.
var errLockHeld = errors.New("lock is held by another process")

// Lock is an advisory lock that keeps concurrent runs of the cli tool from
// doing the same work at the same time.
type Lock struct {
	f    *os.File
	path string
}

// Path returns the path of the lock file.
func (l *Lock) Path() string {
	return l.path
}

// Unlock releases the lock.
func (l *Lock) Unlock() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := unlockFile(l.f)
	if closeErr := l.f.Close(); err == nil {
		err = closeErr
	}
	l.f = nil
	return err
}

// Lock takes the named lock, failing right away with a UserError when
// another process holds it. Lock files live in RuntimeDir(), falling back
// to CacheDir(). Release the lock with Unlock; it is also released when the
// process exits.
func (m *Meta) Lock(name string) (*Lock, error) {
	path, err := m.lockPath(name)
	if err != nil {
		return nil, err
	}

	lock, err := tryLock(path)
	if errors.Is(err, errLockHeld) {
		message := fmt.Sprintf("Another %s process holds the %q lock", valueOr(m.appName, "cli"), name)
		if pid := lockHolder(path); pid != "" {
			message += " (pid " + pid + ")"
		}
		return nil, &UserError{
			Message: message + ".",
			Hint:    "Wait for it to finish, then try again.",
			Err:     err,
		}
	}
	return lock, err
}

// LockWait takes the named lock, waiting until it is free or ctx is done.
func (m *Meta) LockWait(ctx context.Context, name string) (*Lock, error) {
	path, err := m.lockPath(name)
	if err != nil {
		return nil, err
	}

	for {
		lock, err := tryLock(path)
		if !errors.Is(err, errLockHeld) {
			return lock, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

// lockPath returns the path of the named lock file, creating its directory.
func (m *Meta) lockPath(name string) (string, error) {
	dir, err := RuntimeDir(m.appName)
	if err != nil {
		dir, err = CacheDir(m.appName)
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	safe := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r < 0x20 {
			return '_'
		}
		return r
	}, name)
	return filepath.Join(dir, safe+".lock"), nil
}

// tryLock opens path and takes an exclusive lock on it without waiting.
func tryLock(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}

	if err := lockFile(f); err != nil {
		f.Close()
		return nil, err
	}

	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	}
	return &Lock{f: f, path: path}, nil
}

// lockHolder returns the pid recorded in a lock file.
func lockHolder(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	pid := strings.TrimSpace(string(data))
	if _, err := strconv.Atoi(pid); err != nil {
		return ""
	}
	return pid
}
