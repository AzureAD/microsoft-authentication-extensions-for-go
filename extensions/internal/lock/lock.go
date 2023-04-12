// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/internal/flock"
)

// flocker helps tests fake flock
type flocker interface {
	Fh() *os.File
	Path() string
	TryLock() (bool, error)
	Unlock() error
}

// Lock uses a file lock to coordinate access to resources shared with other processes.
// Callers are responsible for preventing races within a process.
type Lock struct {
	f          flocker
	retryDelay time.Duration
	tries      int
}

// New is the constructor for Lock. "p" is the path to the lock file.
func New(p string, tries int, retryDelay time.Duration) (*Lock, error) {
	// ensure all dirs in the path exist before flock tries to create the file
	err := os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &Lock{
		f:          flock.New(p),
		retryDelay: retryDelay,
		tries:      tries,
	}, nil
}

// Lock acquires the file lock on behalf of this process. On some platforms the lock isn't
// mandatory or even exclusive within a process. For example, on Linux the lock is advisory
// and the kernel may allow multiple threads to acquire it simultaneously.
func (l *Lock) Lock(ctx context.Context) error {
	for i := 0; i < l.tries; i++ {
		locked, err := l.f.TryLock()
		if err != nil {
			return err
		}
		if locked {
			// TODO: do we really need this? Exposing the file handle seems to be the only reason to vendor flock
			_ , _ = l.f.Fh().WriteString(fmt.Sprintf("{%d} {%s}", os.Getpid(), os.Args[0]))
			return nil
		}
		// don't delay after the final retry
		if i < l.tries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(l.retryDelay):
				// retry
			}
		}
	}
	return errors.New("couldn't acquire file lock")
}

// Unlock releases the lock and deletes the lock file.
func (l *Lock) Unlock() error {
	err := l.f.Unlock()
	if err == nil {
		err = os.Remove(l.f.Path())
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
