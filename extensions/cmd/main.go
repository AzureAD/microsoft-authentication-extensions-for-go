// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions"
	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor"
	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor/file"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

var f = flag.Bool("f", false, "use file accessor")

func main() {
	flag.Parse()
	wd := os.TempDir()
	var a accessor.Cache
	var err error
	if *f {
		var fh *os.File
		fh, err = os.CreateTemp(wd, "msalext")
		if err != nil {
			panic(err)
		}
		fmt.Printf("created cache file %s\n", fh.Name())
		err = fh.Close()
		if err != nil {
			panic(err)
		}
		defer remove(fh.Name())
		p := filepath.Join(wd, "msal.cache")
		a, err = file.New(p)
		if err == nil {
			defer remove(p)
		}
	} else {
		a, err = accessor.New("name")
	}
	if err != nil {
		panic(err)
	}
	ts := filepath.Join(wd, "timestamp")
	tc, err := extensions.NewTokenCache(a, ts)
	if err != nil {
		panic(err)
	}
	defer remove(ts)
	c := testCache([]byte("data"))
	err = tc.Export(context.Background(), &c, cache.ExportHints{})
	if err != nil {
		panic(err)
	}
	err = tc.Replace(context.Background(), &c, cache.ReplaceHints{})
	if err != nil {
		panic(err)
	}
}

func remove(p string) {
	if err := os.Remove(p); err != nil {
		fmt.Printf("couldn't delete %s: %v\n", p, err)
	} else {
		fmt.Printf("deleted %s\n", p)
	}
}

type testCache []byte

func (t *testCache) Marshal() ([]byte, error) {
	return *t, nil
}

func (t *testCache) Unmarshal(b []byte) error {
	cp := make([]byte, len(b))
	copy(cp, b)
	*t = cp
	return nil
}
