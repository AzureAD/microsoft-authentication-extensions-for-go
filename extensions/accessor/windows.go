// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build windows
// +build windows

package accessor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/billgraziano/dpapi"
)

type Accessor struct {
	p string
	m *sync.RWMutex
}

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
	return &Accessor{p: p, m: &sync.RWMutex{}}, err
}

func (w *Accessor) Read(context.Context) ([]byte, error) {
	w.m.RLock()
	defer w.m.RUnlock()

	data, err := os.ReadFile(w.p)
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		data, err = dpapi.DecryptBytes(data)
	}
	return data, err
}

func (w *Accessor) Write(ctx context.Context, data []byte) error {
	w.m.Lock()
	defer w.m.Unlock()

	data, err := dpapi.EncryptBytes(data)
	if err == nil {
		err = os.WriteFile(w.p, data, 0600)
	}
	return err
}

var _ Cache = (*Accessor)(nil)
