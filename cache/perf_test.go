// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package cache

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/cache/accessor/file"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/stretchr/testify/require"
)

// this file benchmarks MSAL clients using Cache and file.Accessor

func newCache(b *testing.B) *Cache {
	p := filepath.Join(b.TempDir(), b.Name())
	a, err := file.New(p)
	require.NoError(b, err)
	c, err := New(a, p+".timestamp")
	require.NoError(b, err)
	return c
}

func BenchmarkConfidentialClient(b *testing.B) {
	for _, baseline := range []bool{false, true} {
		name := "file accessor"
		if baseline {
			name = "no persistence"
		}
		b.Run(name, func(b *testing.B) {
			var c cache.ExportReplace
			if !baseline {
				c = newCache(b)
			}
			cred, err := confidential.NewCredFromSecret("*")
			require.NoError(b, err)
			client, err := confidential.New(
				"https://login.microsoftonline.com/tenant", "ID", cred, confidential.WithCache(c), confidential.WithHTTPClient(&mockSTS{}),
			)
			require.NoError(b, err)

			gr := 10
			wg := sync.WaitGroup{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for i := 0; i < gr; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						s := fmt.Sprint(n)
						_, _ = client.AcquireTokenByCredential(ctx, []string{s})
						_, _ = client.AcquireTokenSilent(ctx, []string{s})
					}(i)
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkPublicClient(b *testing.B) {
	for _, baseline := range []bool{false, true} {
		name := "file accessor"
		if baseline {
			name = "no persistence"
		}
		b.Run(name, func(b *testing.B) {
			var c cache.ExportReplace
			if !baseline {
				c = newCache(b)
			}
			client, err := public.New("clientID", public.WithCache(c), public.WithHTTPClient(&mockSTS{}))
			require.NoError(b, err)

			gr := 10
			wg := sync.WaitGroup{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for i := 0; i < gr; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						s := fmt.Sprint(n)
						ar, _ := client.AcquireTokenByUsernamePassword(ctx, []string{s}, s, "password")
						_, _ = client.AcquireTokenSilent(ctx, []string{s}, public.WithSilentAccount(ar.Account))
					}(i)
				}
				wg.Wait()
			}
		})
	}
}
