// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build (darwin && cgo) || (linux && cgo) || windows
// +build darwin,cgo linux,cgo windows

package accessor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const msalextManualTest = "MSALEXT_MANUAL_TEST"

var (
	ctx = context.Background()

	// the Windows implementation doesn't require user interaction
	manualTests = runtime.GOOS == "windows" || os.Getenv(msalextManualTest) != ""
)

func TestReadWrite(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	for _, test := range []struct {
		desc string
		want []byte
	}{
		{desc: "Test when no stored data exists"},
		{desc: "Test writing data then reading it", want: []byte("want")},
	} {
		t.Run(test.desc, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), t.Name())
			a, err := New(p)
			require.NoError(t, err)

			if test.want != nil {
				cp := make([]byte, len(test.want))
				copy(cp, test.want)
				err = a.Write(ctx, cp)
				require.NoError(t, err)
			}

			actual, err := a.Read(ctx)
			require.NoError(t, err)
			require.Equal(t, test.want, actual)
		})
	}
}

func TestEmpty(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	s := t.Name()
	if runtime.GOOS != "darwin" {
		s = filepath.Join(t.TempDir(), s)
	}
	a, err := New(s)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Nil(t, actual)
}

func TestRace(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	s := t.Name()
	if runtime.GOOS != "darwin" {
		s = filepath.Join(t.TempDir(), s)
	}
	a, err := New(s)
	require.NoError(t, err)

	expected := "expected"
	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !t.Failed() {
				actual := []byte{}
				err := a.Write(ctx, []byte(expected))
				if err == nil {
					actual, err = a.Read(ctx)
				}
				if err != nil {
					t.Error(err)
				} else if a := string(actual); a != expected {
					t.Errorf("expected %q, got %q", expected, a)
				}
			}
		}()
	}
	wg.Wait()
}

func TestRoundTrip(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	s := t.Name()
	if runtime.GOOS != "darwin" {
		s = filepath.Join(t.TempDir(), s)
	}
	a, err := New(s)
	require.NoError(t, err)

	expected := []byte("expected2")
	err = a.Write(ctx, expected)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
