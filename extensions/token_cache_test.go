// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package extensions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// fakeExternalCache implements accessor.Cache to fake a persistent cache
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
	cp := make([]byte, len(b))
	copy(cp, b)
	a.data = cp
	return err
}

// fakeInternalCache implements cache.Un/Marshaler to fake an MSAL application's in-memory cache
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
	err error
}

func (l fakeLock) Lock(context.Context) error {
	return l.err
}

func (l fakeLock) Unlock() error {
	return l.err
}

func TestCreatesTimestampFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "newdir", t.Name())
	_, err := NewTokenCache(&fakeExternalCache{}, p)
	require.NoError(t, err)
	_, err = os.Stat(p)
	require.NoError(t, err, "NewTokenCache should have created the timestamp file")
}

func TestExport(t *testing.T) {
	ec := fakeExternalCache{}
	ic := &fakeInternalCache{}
	p := filepath.Join(t.TempDir(), t.Name())
	tc, err := NewTokenCache(&ec, p)
	require.NoError(t, err)
	f, err := os.Stat(p)
	require.NoError(t, err, "NewTokenCache should have created the timestamp file")
	lastWrite := f.ModTime()

	// Export should write the in-memory cache to the accessor and touch the timestamp file
	for i := 0; i < 3; i++ {
		s := fmt.Sprint(i)
		*ic = fakeInternalCache{data: []byte(s)}
		err = tc.Export(ctx, ic, cache.ExportHints{})
		require.NoError(t, err)
		require.Equal(t, []byte(s), ec.data)

		f, err = os.Stat(p)
		require.NoError(t, err)
		mt := f.ModTime()
		require.NotEqual(t, lastWrite, mt, "Export should have updated the timestamp")
		lastWrite = mt
	}
}

func TestLockError(t *testing.T) {
	tc, err := NewTokenCache(&fakeExternalCache{}, filepath.Join(t.TempDir(), t.Name()))
	require.NoError(t, err)
	expected := errors.New("expected")
	tc.l = fakeLock{err: expected}
	err = tc.Export(ctx, &fakeInternalCache{}, cache.ExportHints{})
	require.EqualError(t, err, expected.Error())
}

func TestRace(t *testing.T) {
	ic := fakeInternalCache{}
	ec := fakeExternalCache{}
	tc, err := NewTokenCache(&ec, filepath.Join(t.TempDir(), t.Name()))
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if !t.Failed() {
				err := tc.Replace(ctx, &ic, cache.ReplaceHints{})
				if err == nil {
					err = tc.Export(ctx, &ic, cache.ExportHints{})
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
	ec := &fakeExternalCache{}
	p := filepath.Join(t.TempDir(), t.Name())
	_, err := os.Stat(p)
	require.Error(t, err, "timestamp file shouldn't exist yet")
	tc, err := NewTokenCache(ec, p)
	require.NoError(t, err)
	require.Empty(t, ic)

	// Replace should read data from the accessor into the in-memory cache, observing the timestamp file
	s := ""
	for i := 0; i < 4; i++ {
		s = fmt.Sprint(i)
		*ec = fakeExternalCache{data: []byte(s)}
		err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
		require.NoError(t, err)
		require.EqualValues(t, []byte(s), ic.data)
		// touch the timestamp file to indicate another accessor wrote data. Backdating ensures the
		// timestamp changes between iterations even when one executes faster than file time resolution
		tm := time.Now().Add(time.Duration(-i) * time.Second)
		require.NoError(t, os.Chtimes(p, tm, tm))
	}

	// Replace should return in memory data when the timestamp indicates no intervening write to the persistent cache
	expected := []byte(s)
	for i := 0; i < 4; i++ {
		err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
		require.NoError(t, err)
		require.EqualValues(t, expected, ic.data)
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
		tc, err := NewTokenCache(ec, p)
		require.NoError(t, err)

		err = tc.Replace(ctx, &fakeInternalCache{}, cache.ReplaceHints{})
		require.Equal(t, expected, err)
	})

	for _, transient := range []bool{true, false} {
		name := "unmarshal error"
		if transient {
			name = "transient " + name
		}
		t.Run(name, func(t *testing.T) {
			ums := 0
			ic := fakeInternalCache{unmarshalCallback: func() error {
				ums++
				if transient && ums > 1 {
					return nil
				}
				return expected
			}}
			ec := &fakeExternalCache{}

			p := filepath.Join(t.TempDir(), t.Name())
			tc, err := NewTokenCache(ec, p)
			require.NoError(t, err)

			err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
			if transient {
				require.NoError(t, err)
				require.Equal(t, 2, ums)
			} else {
				require.Equal(t, expected, err)
			}
		})
	}
}

func TestReplaceTimestamp(t *testing.T) {
	startData := []byte("starting data")
	ec := fakeExternalCache{data: startData}
	ic := fakeInternalCache{}
	p := filepath.Join(t.TempDir(), t.Name())
	tc, err := NewTokenCache(&ec, p)
	require.NoError(t, err)

	err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
	require.NoError(t, err)
	require.Equal(t, startData, ic.data)

	ec.readCallback = func() error {
		t.Fatal("timestamp didn't change but TokenCache called Read")
		return nil
	}
	err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
	require.NoError(t, err)
	require.Equal(t, startData, ic.data)

	ec.data = append(ec.data, []byte(" + new data")...)
	ec.readCallback = nil
	require.NoError(t, os.Remove(p), "failed to remove timestamp file")
	err = tc.Replace(ctx, &ic, cache.ReplaceHints{})
	require.NoError(t, err)
	require.Equal(t, ec.data, ic.data, "timestamp was missing but TokenCache didn't call Read")
}
