// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

// TODO: add other platforms
//go:build windows
// +build windows

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
	runTests = runtime.GOOS == "windows" || os.Getenv(msalextManualTest) != ""
)

func TestRace(t *testing.T) {
	if !runTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Empty(t, actual)

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
	if !runTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Empty(t, actual)

	expected := []byte("expected")
	err = a.Write(ctx, expected)
	require.NoError(t, err)

	actual, err = a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
