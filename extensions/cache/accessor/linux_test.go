// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build linux
// +build linux

package accessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTooManyAttributes(t *testing.T) {
	_, err := New(t.Name(), WithAttribute("", ""), WithAttribute("", ""), WithAttribute("", ""))
	require.Error(t, err)
}

func TestWithAttribute(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	a, err := New(t.Name())
	require.NoError(t, err)
	require.Empty(t, a.attributes)

	expected := []byte("expected")
	err = a.Write(ctx, expected)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	b, err := New(t.Name(), WithAttribute("1", "1"))
	require.NoError(t, err)
	require.Equal(t, 1, len(b.attributes))
	actual, err = b.Read(ctx)
	require.NoError(t, err)
	require.Empty(t, actual)

	c, err := New(t.Name(), WithAttribute("1", "1"), WithAttribute("2", "2"))
	require.NoError(t, err)
	require.Equal(t, 2, len(c.attributes))
	actual, err = c.Read(ctx)
	require.NoError(t, err)
	require.Empty(t, actual)
}

func TestWithLabel(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	label := "label"
	a, err := New(t.Name(), WithLabel(label))
	require.NoError(t, err)
	require.Equal(t, label, a.label)

	expected := []byte("expected")
	err = a.Write(ctx, expected)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
