// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package cache_test

import (
	"github.com/AzureAD/microsoft-authentication-extensions-for-go/cache"
	"github.com/AzureAD/microsoft-authentication-extensions-for-go/cache/accessor"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// This example shows how to configure an MSAL public client to store data in a peristent, encrypted cache.
func Example() {
	// On Linux and macOS, "s" is an arbitrary name identifying the cache.
	// On Windows, it's the path to a file in which to store cache data.
	s := "..."
	a, err := accessor.New(s)
	if err != nil {
		// TODO: handle error
	}
	c, err := cache.New(a, s)
	if err != nil {
		// TODO: handle error
	}
	app, err := public.New("client-id", public.WithCache(c))
	if err != nil {
		// TODO: handle error
	}

	_ = app
}
