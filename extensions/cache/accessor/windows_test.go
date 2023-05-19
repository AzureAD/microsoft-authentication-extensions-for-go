// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build windows
// +build windows

package accessor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteEncryption(t *testing.T) {
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := New(p)
	require.NoError(t, err)

	data := []byte(`{"key":"value"}`)
	require.NoError(t, json.Unmarshal(data, &struct{}{}), "test bug: data should unmarshal")
	require.NoError(t, a.Write(ctx, data))

	// Write should have encrypted data before writing it to the file
	actual, err := os.ReadFile(p)
	require.NoError(t, err)
	require.NotEmpty(t, actual)
	err = json.Unmarshal(actual, &struct{}{})
	require.Error(t, err, "Unmarshal should fail because the file's content, being encrypted, isn't JSON")
}
