// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package lock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

type fakeFlock struct {
	err error
	p   string
}

func (f fakeFlock) Fh() *os.File {
	fh, _ := os.Open(f.p)
	return fh
}

func (f fakeFlock) Path() string {
	return f.p
}

func (f fakeFlock) TryLockContext(context.Context, time.Duration) (bool, error) {
	return f.err == nil, f.err
}

func (f fakeFlock) Unlock() error {
	return f.err
}

func TestCreatesAndRemovesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nonexistent", t.Name())
	lock, err := New(p, 0)
	require.NoError(t, err)
	require.NoFileExists(t, p)

	err = lock.Lock(ctx)
	require.NoError(t, err)
	require.FileExists(t, p, "Lock didn't create the file")

	err = lock.Unlock()
	require.NoError(t, err)
	require.NoFileExists(t, p, "Unlock didn't remove the file")
}

func TestLockError(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	lock, err := New(p, 0)
	require.NoError(t, err)
	expected := errors.New("expected")
	lock.f = fakeFlock{err: expected}
	require.Equal(t, lock.Lock(ctx), expected)
}

func TestLockTimeout(t *testing.T) {
	defer func(d time.Duration) { timeout = d }(timeout)
	timeout = 0

	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p, 0)
	require.NoError(t, err)
	err = a.Lock(ctx)
	require.NoError(t, err)
	b, err := New(p, 0)
	require.NoError(t, err)

	err = b.Lock(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, a.Unlock())
}

func TestUnlockErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	lock, err := New(p, 0)
	require.NoError(t, err)

	err = lock.Lock(ctx)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		// Remove would fail on Windows because the file lock is mandatory there
		require.NoError(t, os.Remove(p))
	}
	// Unlock should return nil even when the lock file has been removed
	require.NoError(t, lock.Unlock())

	expected := errors.New("it didn't work")
	lock.f = fakeFlock{err: expected}
	actual := lock.Unlock()
	require.Equal(t, expected, actual)
}
