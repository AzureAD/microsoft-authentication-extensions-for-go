// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build !(darwin || linux || windows) || (linux && !cgo)
// +build !darwin,!linux,!windows linux,!cgo

package accessor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

var er error = fmt.Errorf("no implementation for %s", runtime.GOOS)

func init() {
	if runtime.GOOS == "linux" {
		er = errors.New("building Accessor requires cgo on linux")
	}
}

// Accessor isn't supported in this environment
type Accessor struct{}

// New returns an error because Accessor isn't supported in this environment
func New(string) (*Accessor, error) {
	return nil, er
}

// Read returns an error because Accessor isn't supported in this environment
func (Accessor) Read(context.Context) ([]byte, error) {
	return nil, er
}

// Write returns an error because Accessor isn't supported in this environment
func (Accessor) Write(context.Context, []byte) error {
	return er
}

var _ Cache = (*Accessor)(nil)
