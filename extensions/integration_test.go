// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package extensions

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor/file"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/stretchr/testify/require"
)

func TestConfidentialClient(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := file.New(p)
	require.NoError(t, err)
	c, err := NewTokenCache(a, p+".timestamp")
	require.NoError(t, err)
	cred, err := confidential.NewCredFromSecret("*")
	require.NoError(t, err)
	client, err := confidential.New(
		"https://login.microsoftonline.com/tenant", "clientID", cred, confidential.WithCache(c), confidential.WithHTTPClient(&mockSTS{}),
	)
	require.NoError(t, err)

	gr := 2
	wg := sync.WaitGroup{}
	for i := 0; i < gr; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if t.Failed() {
				return
			}
			s := fmt.Sprint(n)
			ar, err := client.AcquireTokenByCredential(ctx, []string{s})
			if err != nil {
				t.Error(err)
			} else if ar.AccessToken != s {
				t.Errorf("possible test bug: expected %q from STS, got %q", s, ar.AccessToken)
			} else {
				ar, err = client.AcquireTokenSilent(ctx, []string{s})
				if err != nil {
					t.Error(err)
				} else if ar.AccessToken != s {
					t.Errorf("possible cache corruption: expected %q, got %q", s, ar.AccessToken)
				}
			}
		}(i)
	}
	wg.Wait()

	// cache should have an access token from each goroutine
	lost := gr
	for i := 0; i < gr; i++ {
		s := fmt.Sprint(i)
		ar, err := client.AcquireTokenSilent(ctx, []string{s})
		if err == nil {
			lost--
			if ar.AccessToken != s {
				t.Errorf("possible cache corruption: expected %q, got %q", s, ar.AccessToken)
			}
		}
	}
	require.Equal(t, 0, lost, "lost %d of %d tokens", lost, gr)
}

func TestPublicClient(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), t.Name())
	a, err := file.New(p)
	require.NoError(t, err)
	c, err := NewTokenCache(a, p+".timestamp")
	require.NoError(t, err)
	sts := mockSTS{}
	client, err := public.New("clientID", public.WithCache(c), public.WithHTTPClient(&sts))
	require.NoError(t, err)

	gr := 2
	wg := sync.WaitGroup{}
	for i := 0; i < gr; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if t.Failed() {
				return
			}
			s := fmt.Sprint(n)
			ar, err := client.AcquireTokenByUsernamePassword(ctx, []string{s}, s, "password")
			if err != nil {
				t.Error(err)
			} else if ar.AccessToken != s {
				t.Errorf("possible test bug: expected %q from STS, got %q", s, ar.AccessToken)
			} else {
				ar, err = client.AcquireTokenSilent(ctx, []string{s}, public.WithSilentAccount(ar.Account))
				if err != nil {
					t.Error(err)
				} else if ar.AccessToken != s {
					t.Errorf("possible cache corruption: expected %q, got %q", s, ar.AccessToken)
				}
			}
		}(i)
	}
	wg.Wait()

	accounts, err := client.Accounts(ctx)
	require.NoError(t, err)
	require.Equal(t, gr, len(accounts), "should have a cached account for each goroutine")

	// Verify no access token cached above was lost due to a race. Silent auth should return a cached
	// access token given any scope above. A token request during this loop indicates the client
	// exchanged a refresh token for the access token it should have found in the cache.
	lostATs, reqs := 0, 0
	sts.tokenRequestCallback = func(*http.Request) { reqs++ }
	for _, a := range accounts {
		s, _, found := strings.Cut(a.HomeAccountID, ".")
		require.True(t, found, "unexpected home account ID %q", a.HomeAccountID)
		ar, err := client.AcquireTokenSilent(ctx, []string{s}, public.WithSilentAccount(a))
		if err != nil {
			// the cache has no access token for the expected scope and no refresh token for the account
			lostATs++
		} else if ar.AccessToken != s {
			t.Errorf("possible cache corruption: expected %q, got %q", s, ar.AccessToken)
		}
	}
	require.Equal(t, 0, lostATs+reqs, "lost %d of %d access tokens", reqs, gr)

	// The cache has all the expected access tokens but may have lost refresh tokens, so we try silent
	// auth again for each account, passing a new scope to force the client to use a refresh token.
	lostRTs := 0
	for _, a := range accounts {
		s := "novelscope"
		ar, err := client.AcquireTokenSilent(ctx, []string{s}, public.WithSilentAccount(a))
		if err != nil {
			lostRTs++
		} else if ar.AccessToken != s {
			t.Errorf("possible cache corruption: expected %q, got %q", s, ar.AccessToken)
		}
	}
	require.Equal(t, 0, lostRTs, "lost %d of %d refresh tokens", lostRTs, gr)
}
