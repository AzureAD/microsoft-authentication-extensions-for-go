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

func TestReadWriteClear(t *testing.T) {
	for _, test := range []struct {
		desc              string
		initialData, want []byte
	}{
		{desc: "Test when the file exists", initialData: []byte("data"), want: []byte("want")},
		{desc: "Test when the file doesn't exist", want: []byte("want")},
	} {
		t.Run(test.desc, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), t.Name())
			if test.initialData != nil {
				require.NoError(t, os.MkdirAll(filepath.Dir(p), 0700))
				f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
				require.NoError(t, err)
				_, err = f.Write(test.initialData)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			a, err := New(p)
			require.NoError(t, err)

			if test.initialData != nil {
				actual, err := a.Read(ctx)
				require.NoError(t, err)
				require.Equal(t, test.initialData, actual)
			}

			cp := make([]byte, len(test.want))
			copy(cp, test.want)
			err = a.Write(ctx, cp)
			require.NoError(t, err)

			actual, err := a.Read(ctx)
			require.NoError(t, err)
			require.Equal(t, test.want, actual)

			require.NoError(t, a.Clear(context.Background()))
		})
	}
}
