// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build windows
// +build windows

package accessor

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Storage stores data in a file encrypted by the Windows data protection API.
type Storage struct {
	m *sync.RWMutex
	p string
}

// New is the constructor for Storage. "p" is the path to the file in which to store data.
func New(p string) (*Storage, error) {
	return &Storage{m: &sync.RWMutex{}, p: p}, nil
}

// Read returns data from the file. If the file doesn't exist, Read returns a nil slice and error.
func (s *Storage) Read(context.Context) ([]byte, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	data, err := os.ReadFile(s.p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		data, err = dpapi(decrypt, data)
	}
	return data, err
}

// Write stores data in the file, creating the file if it doesn't exist.
func (s *Storage) Write(ctx context.Context, data []byte) error {
	s.m.Lock()
	defer s.m.Unlock()

	data, err := dpapi(encrypt, data)
	if err != nil {
		return err
	}
	err = os.WriteFile(s.p, data, 0600)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(s.p)
		if err = os.MkdirAll(dir, 0700); err == nil {
			err = os.WriteFile(s.p, data, 0600)
		}
	}
	return err
}

type operation int

const (
	decrypt operation = iota
	encrypt
)

func dpapi(op operation, data []byte) (result []byte, err error) {
	out := windows.DataBlob{}
	defer func() {
		if out.Data != nil {
			_, e := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
			// prefer returning DPAPI errors because they're more interesting than LocalFree errors
			if e != nil && err == nil {
				err = e
			}
		}
	}()
	in := windows.DataBlob{Data: &data[0], Size: uint32(len(data))}
	switch op {
	case decrypt:
		// https://learn.microsoft.com/windows/win32/api/dpapi/nf-dpapi-cryptunprotectdata
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, 1, &out)
	case encrypt:
		// https://learn.microsoft.com/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, 1, &out)
	default:
		err = errors.New("invalid operation")
	}
	if err == nil {
		// cast out.Data to a pointer to an arbitrarily long array, then slice the array and copy out.Size bytes from the
		// slice to result. This avoids allocating memory for a throwaway buffer but imposes a max size on the data because
		// the fictive array backing the slice can't be larger than the address space or the maximum value of an int. Those
		// values vary by platform, so the array size here is a compromise for 32-bit systems and allows ~2 GB of data.
		result = make([]byte, out.Size)
		source := (*[math.MaxInt32 - 1]byte)(unsafe.Pointer(out.Data))[:]
		copy(result, source)
	}
	return result, err
}