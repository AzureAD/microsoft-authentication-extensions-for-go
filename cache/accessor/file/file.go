// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/cache/accessor"
)

// Storage stores data in an unencrypted file.
type Storage struct {
	m *sync.RWMutex
	p string
}

// New is the constructor for Storage. "p" is the path to the file in which to store data.
func New(p string) (*Storage, error) {
	return &Storage{m: &sync.RWMutex{}, p: p}, nil
}

// Read returns the file's content or, if the file doesn't exist, a nil slice and error.
func (s *Storage) Read(context.Context) ([]byte, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	b, err := os.ReadFile(s.p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

// Write stores data in the file, overwriting any content, and creates the file if necessary.
func (s *Storage) Write(ctx context.Context, data []byte) error {
	s.m.Lock()
	defer s.m.Unlock()
	err := os.WriteFile(s.p, data, 0600)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(s.p)
		if err = os.MkdirAll(dir, 0700); err == nil {
			err = os.WriteFile(s.p, data, 0600)
		}
	}
	return err
}

var _ accessor.Accessor = (*Storage)(nil)
