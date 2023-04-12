// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor"
)

// Accessor stores data in an unencrypted file.
type Accessor struct {
	p string
	m *sync.Mutex
}

// New is the constructor for Accessor. "p" is the path of the file Accessor should store data in.
// New will create this file, if it doesn't exist.
func New(p string) (*Accessor, error) {
	_, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(p)
		err = os.MkdirAll(dir, os.ModePerm)
		if err == nil {
			var f *os.File
			if f, err = os.Create(p); err == nil {
				err = f.Close()
			}
		}
	}
	return &Accessor{p, &sync.Mutex{}}, err
}

// Read returns data from the file.
func (a *Accessor) Read(context.Context) ([]byte, error) {
	a.m.Lock()
	defer a.m.Unlock()
	return os.ReadFile(a.p)
}

// Write stores data in the file.
func (a *Accessor) Write(ctx context.Context, data []byte) error {
	a.m.Lock()
	defer a.m.Unlock()
	return os.WriteFile(a.p, data, os.ModePerm)
}

var _ accessor.Cache = (*Accessor)(nil)
