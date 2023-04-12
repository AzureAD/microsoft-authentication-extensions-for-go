// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package extensions

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/accessor"
	"github.com/AzureAD/microsoft-authentication-extensions-for-go/extensions/internal/lock"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// retryDelay lets tests prevent delays when faking errors in Replace.
// The default value allows Replace 1.5 seconds to read valid cache data.
var retryDelay = 20 * time.Millisecond

// locker helps tests fake Lock
type locker interface {
	Lock(context.Context) error
	Unlock() error
}

// TokenCache caches authentication data in external storage, coordinated by a file lock.
type TokenCache struct {
	// accessor provides read/write access to an external data store
	accessor accessor.Cache
	// data is accessor's data as of the last sync
	data []byte
	// l coordinates with other processes
	l locker
	// m coordinates this process's goroutines
	m *sync.Mutex
	// sync is the timestamp when data was last written to or read from accessor
	sync time.Time
	// ts is the path to a file used to timestamp Export and Replace operations
	ts string
}

// NewTokenCache is the constructor for TokenCache. "p" is the path to a file used to track
// when external cache data changes. NewTokenCache will create this file and any directories
// in its path which don't already exist.
func NewTokenCache(c accessor.Cache, p string) (*TokenCache, error) {
	err := os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if err != nil {
		return nil, err
	}
	lock, err := lock.New(p+".lockfile", 60, time.Millisecond)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	err = f.Close()
	return &TokenCache{accessor: c, l: lock, m: &sync.Mutex{}, ts: p}, err
}

// Export writes the bytes marshaled by "m" to the accessor.
func (t *TokenCache) Export(ctx context.Context, m cache.Marshaler, h cache.ExportHints) error {
	t.m.Lock()
	defer t.m.Unlock()

	data, err := m.Marshal()
	if err != nil {
		return err
	}
	err = t.l.Lock(ctx)
	if err != nil {
		return err
	}
	defer func() {
		e := t.l.Unlock()
		if err == nil {
			err = e
		}
	}()
	if err = t.accessor.Write(ctx, data); err == nil {
		// touch the timestamp file to record the time of this write; discard any
		// error because this is just an optimization to avoid redundant reads
		t.sync = time.Now()
		_ = os.Chtimes(t.ts, t.sync, t.sync)
		t.data = data
	}
	return err
}

// Replace reads bytes from the accessor and unmarshals them to "u".
func (t *TokenCache) Replace(ctx context.Context, u cache.Unmarshaler, h cache.ReplaceHints) error {
	t.m.Lock()
	defer t.m.Unlock()

	// If the timestamp file indicates cached data hasn't changed since we last read or wrote it,
	// return t.data, which is the data as of that time. Discard any error from reading the timestamp
	// because this is just an optimization to prevent unnecessary reads. If we don't know whether
	// cached data has changed, we assume it has.
	read := true
	data := t.data
	f, err := os.Stat(t.ts)
	if err == nil {
		mt := f.ModTime()
		read = !mt.Equal(t.sync)
	}
	// Unmarshal the accessor's data, reading it first if needed. We don't acquire the file lock before
	// reading from the accessor because it isn't strictly necessary and is relatively expensive. In the
	// unlikely event that this read overlaps with a write and returns malformed data, we'll get an error
	// from Unmarshal and try again.
	tries := 75
	for i := 0; i < tries; i++ {
		if read {
			data, err = t.accessor.Read(ctx)
			if err != nil {
				break
			}
		}
		err = u.Unmarshal(data)
		if err == nil {
			break
		}
		// don't wait after the final try
		if i < tries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// got an error from Unmarshal: wait, try again
			}
		}
	}
	// Update the sync time only if we read from the accessor and unmarshaled its data. Otherwise
	// the data hasn't changed since the last read/write, or reading failed and we'll try again on
	// the next call.
	if err == nil && read {
		t.data = data
		if f, err := os.Stat(t.ts); err == nil {
			t.sync = f.ModTime()
		}
	}
	return err
}

var _ cache.ExportReplace = (*TokenCache)(nil)
