// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build linux || windows
// +build linux windows

package accessor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
