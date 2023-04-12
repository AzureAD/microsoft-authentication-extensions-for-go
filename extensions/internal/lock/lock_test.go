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

func (f fakeFlock) TryLock() (bool, error) {
	return f.err == nil, f.err
}

func (f fakeFlock) Unlock() error {
	return f.err
}

func TestContention(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p, 1, 0)
	require.NoError(t, err)
	require.NoError(t, a.Lock(ctx))

	// b can't acquire the lock while a holds it
	b, err := New(p, 1, 0)
	require.NoError(t, err)
	require.Error(t, b.Lock(ctx))

	// and vice versa
	require.NoError(t, a.Unlock())
	require.NoError(t, b.Lock(ctx))
	err = a.Lock(ctx)
	require.Error(t, err)

	// but Lock() succeeds on a locked instance
	err = b.Lock(ctx)
	require.NoError(t, err)

	require.NoError(t, b.Unlock())
}

func TestLockError(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	lock, err := New(p, 1, 0)
	require.NoError(t, err)
	expected := errors.New("expected")
	lock.f = fakeFlock{err: expected, p: p}
	require.Equal(t, lock.Lock(ctx), expected)
}

func TestNewCreatesAndRemovesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nonexistent", t.Name())
	lock, err := New(p, 1, 0)
	require.NoError(t, err)
	require.NoFileExists(t, p, "lock file shouldn't exist when lock is unlocked")
	require.NoError(t, lock.Lock(ctx))
	require.FileExists(t, p, "lock file should exist when Lock is locked")
	require.NoError(t, lock.Unlock())
	require.NoFileExists(t, p, "lock file shouldn't exist when Lock is unlocked")
}

func TestNewFileExists(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	f, err := os.Create(p)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	_, err = New(p, 1, 0)
	require.NoError(t, err)
}

func TestUnlockErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	lock, err := New(p, 1, 0)
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
	require.Equal(t, expected, actual, "Unlock should return any unexpected error")
}
