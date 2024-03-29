// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build darwin && cgo
// +build darwin,cgo

package accessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithAccount(t *testing.T) {
	if !manualTests {
		t.Skipf("set %s to run this test", msalextManualTest)
	}
	account := "account"
	a, err := New(t.Name(), WithAccount(account))
	require.NoError(t, err)

	expected := []byte("expected")
	err = a.Write(ctx, expected)
	require.NoError(t, err)

	actual, err := a.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	require.NoError(t, a.Delete(ctx))
}
