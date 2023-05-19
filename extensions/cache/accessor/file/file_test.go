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

func TestRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	expected := []byte("expected")
	require.NoError(t, os.WriteFile(p, expected, 0600))

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nonexistent", t.Name())
	a, err := New(p)
	require.NoError(t, err)

	var expected []byte
	for i := 0; i < 4; i++ {
		actual, err := a.Read(ctx)
		require.NoError(t, err)
		require.Equal(t, expected, actual)

		expected = append(expected, byte(i))
		require.NoError(t, a.Write(ctx, expected))
	}
}

func TestWrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	for _, create := range []bool{true, false} {
		name := "file exists"
		if create {
			name = "new file"
		}
		t.Run(name, func(t *testing.T) {
			if create {
				f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL, 0600)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			a, err := New(p)
			require.NoError(t, err)

			expected := []byte("expected")
			require.NoError(t, a.Write(ctx, expected))

			actual, err := os.ReadFile(p)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}
