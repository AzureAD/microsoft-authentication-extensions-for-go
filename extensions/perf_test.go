// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package extensions

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor/file"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/stretchr/testify/require"
)

// this file benchmarks MSAL clients using a TokenCache and file.Accessor

func BenchmarkConfidentialClient(b *testing.B) {
	p := filepath.Join(b.TempDir(), b.Name())
	a, err := file.New(p)
	require.NoError(b, err)
	c, err := NewTokenCache(a, p+".timestamp")
	require.NoError(b, err)
	cred, err := confidential.NewCredFromSecret("*")
	require.NoError(b, err)
	client, err := confidential.New(
		"https://login.microsoftonline.com/tenant", "clientID", cred, confidential.WithCache(c), confidential.WithHTTPClient(&mockSTS{}),
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
}

func BenchmarkConfidentialClient_NoPersistence(b *testing.B) {
	cred, err := confidential.NewCredFromSecret("*")
	require.NoError(b, err)
	client, err := confidential.New("https://login.microsoftonline.com/tenant", "clientID", cred, confidential.WithHTTPClient(&mockSTS{}))
	require.NoError(b, err)
	gr := 10
	wg := sync.WaitGroup{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < gr; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				s := []string{fmt.Sprint(n)}
				_, _ = client.AcquireTokenByCredential(ctx, s)
				_, _ = client.AcquireTokenSilent(ctx, s)
			}(i)
		}
		wg.Wait()
	}
}

func BenchmarkPublicClient(b *testing.B) {
	p := filepath.Join(b.TempDir(), b.Name())
	a, err := file.New(p)
	require.NoError(b, err)
	c, err := NewTokenCache(a, p+".timestamp")
	require.NoError(b, err)
	sts := mockSTS{}
	client, err := public.New("clientID", public.WithCache(c), public.WithHTTPClient(&sts))
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
}

func BenchmarkPublicClient_NoPersistence(b *testing.B) {
	sts := mockSTS{}
	client, err := public.New("clientID", public.WithHTTPClient(&sts))
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
}
