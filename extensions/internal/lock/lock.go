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

// timeout lets tests set the default amount of time allowed to acquire the lock
var timeout = 5 * time.Second

// flocker helps tests fake flock
type flocker interface {
	Fh() *os.File
	Path() string
	TryLockContext(context.Context, time.Duration) (bool, error)
	Unlock() error
}

// Lock uses a file lock to coordinate access to resources shared with other processes.
// Callers are responsible for preventing races within a process.
type Lock struct {
	f          flocker
	retryDelay time.Duration
}

// New is the constructor for Lock. "p" is the path to the lock file.
func New(p string, retryDelay time.Duration) (*Lock, error) {
	// ensure all dirs in the path exist before flock tries to create the file
	err := os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &Lock{f: flock.New(p), retryDelay: retryDelay}, nil
}

// Lock acquires the file lock on behalf of the process. The behavior of concurrent
// and repeated calls is undefined. For example, Linux may or may not allow goroutines
// scheduled on different threads to hold the lock simultaneously.
func (l *Lock) Lock(ctx context.Context) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	locked, err := l.f.TryLockContext(ctx, l.retryDelay)
	if err != nil {
		return err
	}
	if locked {
		_, _ = l.f.Fh().WriteString(fmt.Sprintf("{%d} {%s}", os.Getpid(), os.Args[0]))
		return nil
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
