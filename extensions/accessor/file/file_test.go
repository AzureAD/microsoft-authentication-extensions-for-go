// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestNewCreatesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nonexistent", t.Name())
	a, err := New(p)
	require.NoError(t, err)
	require.NotNil(t, a)
	require.FileExists(t, p)
}

func TestNewPreservesExistingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	f, err := os.Create(p)
	require.NoError(t, err)

	expected := []byte("expected")
	_, err = f.Write(expected)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	a, err := New(p)
	require.NoError(t, err)

	actual, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual, err = a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	expected := []byte("expected")
	require.NoError(t, os.WriteFile(p, expected, os.ModePerm))

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Empty(t, actual)

	expected := []byte("expected")
	require.NoError(t, a.Write(ctx, expected))

	actual, err = a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestWrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	expected := []byte("expected")
	require.NoError(t, a.Write(ctx, expected))

	actual, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
