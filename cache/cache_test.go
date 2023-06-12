// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// fakeExternalCache implements accessor.Accessor to fake a persistent cache
type fakeExternalCache struct {
	data                        []byte
	readCallback, writeCallback func() error
}

func (a *fakeExternalCache) Read(context.Context) ([]byte, error) {
	var err error
	if a.readCallback != nil {
		err = a.readCallback()
	}
	return a.data, err
}

func (a *fakeExternalCache) Write(ctx context.Context, b []byte) error {
	var err error
	if a.writeCallback != nil {
		err = a.writeCallback()
	}
	if err != nil {
		return err
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	a.data = cp
	return nil
}

// fakeInternalCache implements cache.Un/Marshaler to fake an MSAL client's in-memory cache
type fakeInternalCache struct {
	data                               []byte
	marshalCallback, unmarshalCallback func() error
}

func (t *fakeInternalCache) Marshal() ([]byte, error) {
	var err error
	if t.marshalCallback != nil {
		err = t.marshalCallback()
	}
	return t.data, err
}

func (t *fakeInternalCache) Unmarshal(b []byte) error {
	var err error
	if t.unmarshalCallback != nil {
		err = t.unmarshalCallback()
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	t.data = cp
	return err
}

type fakeLock struct {
	lockErr, unlockErr error
}

func (l fakeLock) Lock(context.Context) error {
	return l.lockErr
}

func (l fakeLock) Unlock() error {
	return l.unlockErr
}

func TestExport(t *testing.T) {
	ec := &fakeExternalCache{}
	ic := &fakeInternalCache{}
	p := filepath.Join(t.TempDir(), t.Name(), "ts")
	c, err := New(ec, p)
	require.NoError(t, err)

	// Export should write the in-memory cache to the accessor and touch the timestamp file
	lastWrite := time.Time{}
	touched := false
	for i := 0; i < 3; i++ {
		s := fmt.Sprint(i)
		*ic = fakeInternalCache{data: []byte(s)}
		err = c.Export(ctx, ic, cache.ExportHints{})
		require.NoError(t, err)
		require.Equal(t, []byte(s), ec.data)

		f, err := os.Stat(p)
		require.NoError(t, err)
		mt := f.ModTime()

		// Two iterations of this loop can run within one unit of system time on Windows, leaving the
		// modtime apparently unchanged even though Export updated it. On Windows we therefore skip
		// the strict test, instead requiring only that the modtime change once during this loop.
		if runtime.GOOS != "windows" {
			require.NotEqual(t, lastWrite, mt, "Export didn't update the timestamp")
		}
		if mt != lastWrite {
			touched = true
		}
		lastWrite = mt
	}
	require.True(t, touched, "Export didn't update the timestamp")
}

func TestFilenameCompat(t *testing.T) {
	// verify Cache uses the same lock file name as would e.g. the Python implementation
	p := filepath.Join(t.TempDir(), t.Name())
	ec := fakeExternalCache{
		// Cache should hold the file lock while calling Write
		writeCallback: func() error {
			require.FileExists(t, p+".lockfile", "missing expected lock file")
			return nil
		},
	}
	c, err := New(&ec, p)
	require.NoError(t, err)

	err = c.Export(ctx, &fakeInternalCache{}, cache.ExportHints{})
	require.NoError(t, err)
}

func TestLockError(t *testing.T) {
	c, err := New(&fakeExternalCache{}, filepath.Join(t.TempDir(), t.Name()))
	require.NoError(t, err)
	expected := errors.New("expected")
	c.l = fakeLock{lockErr: expected}
	err = c.Export(ctx, &fakeInternalCache{}, cache.ExportHints{})
	require.EqualError(t, err, expected.Error())
}

func TestPreservesTimestampFileContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	expected := []byte("expected")
	err := os.WriteFile(p, expected, 0600)
	require.NoError(t, err)

	ec := fakeExternalCache{}
	c, err := New(&ec, p)
	require.NoError(t, err)

	ic := fakeInternalCache{data: []byte("data")}
	err = c.Export(ctx, &ic, cache.ExportHints{})
	require.NoError(t, err)
	require.Equal(t, ic.data, ec.data)

	actual, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, expected, actual, "Cache truncated, or wrote to, the timestamp file")
}

func TestRace(t *testing.T) {
	ic := fakeInternalCache{}
	ec := fakeExternalCache{}
	c, err := New(&ec, filepath.Join(t.TempDir(), t.Name()))
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if !t.Failed() {
				err := c.Replace(ctx, &ic, cache.ReplaceHints{})
				if err == nil {
					err = c.Export(ctx, &ic, cache.ExportHints{})
				}
				if err != nil {
					t.Errorf("%d: %s", i, err)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestReplace(t *testing.T) {
	ic := fakeInternalCache{}
	ec := fakeExternalCache{}
	p := filepath.Join(t.TempDir(), t.Name())

	c, err := New(&ec, p)
	require.NoError(t, err)
	require.Empty(t, ic)

	// Replace should read data from the accessor (external cache) into the in-memory cache, observing the timestamp file
	f, err := os.Create(p)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	for i := uint8(0); i < 4; i++ {
		ec.data = []byte{i}
		err = c.Replace(ctx, &ic, cache.ReplaceHints{})
		require.NoError(t, err)
		require.EqualValues(t, ec.data, ic.data)
		// touch the timestamp file to indicate another accessor wrote data. Backdating ensures the
		// timestamp changes between iterations even when one executes faster than file time resolution
		tm := time.Now().Add(-time.Duration(i+1) * time.Second)
		require.NoError(t, os.Chtimes(p, tm, tm))
	}

	// Replace should return in-memory data when the timestamp indicates no intervening write to the persistent cache
	for i := 0; i < 4; i++ {
		err = c.Replace(ctx, &ic, cache.ReplaceHints{})
		require.NoError(t, err)
		// ec.data hasn't changed; ic.data shouldn't change either
		require.EqualValues(t, ec.data, ic.data)
	}
}

func TestReplaceErrors(t *testing.T) {
	realDelay := retryDelay
	retryDelay = 0
	t.Cleanup(func() { retryDelay = realDelay })
	expected := errors.New("expected")

	t.Run("read", func(t *testing.T) {
		ec := &fakeExternalCache{readCallback: func() error {
			return expected
		}}
		p := filepath.Join(t.TempDir(), t.Name())
		c, err := New(ec, p)
		require.NoError(t, err)

		err = c.Replace(ctx, &fakeInternalCache{}, cache.ReplaceHints{})
		require.Equal(t, expected, err)
	})

	for _, transient := range []bool{true, false} {
		name := "unmarshal error"
		if transient {
			name = "transient " + name
		}
		t.Run(name, func(t *testing.T) {
			tries := 0
			ic := fakeInternalCache{unmarshalCallback: func() error {
				tries++
				if transient && tries > 1 {
					return nil
				}
				return expected
			}}
			ec := &fakeExternalCache{}

			p := filepath.Join(t.TempDir(), t.Name())
			c, err := New(ec, p)
			require.NoError(t, err)

			cx, cancel := context.WithTimeout(ctx, time.Millisecond)
			defer cancel()
			err = c.Replace(cx, &ic, cache.ReplaceHints{})
			// err should be nil if the unmarshaling error was transient, non-nil if it wasn't
			require.Equal(t, transient, err == nil)
		})
	}
}

func TestUnlockError(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a := fakeExternalCache{}
	c, err := New(&a, p)
	require.NoError(t, err)

	// Export should return an error from Unlock()...
	unlockErr := errors.New("unlock error")
	c.l = fakeLock{unlockErr: unlockErr}
	err = c.Export(ctx, &fakeInternalCache{}, cache.ExportHints{})
	require.Equal(t, unlockErr, err)

	// ...unless another of its calls returned an error
	writeErr := errors.New("write error")
	a.writeCallback = func() error { return writeErr }
	err = c.Export(ctx, &fakeInternalCache{}, cache.ExportHints{})
	require.Equal(t, writeErr, err)
}
