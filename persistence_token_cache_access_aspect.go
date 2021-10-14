//go:build windows
// +build windows

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/internal"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type FileTokenCache struct {
	lock                               CrossPlatLock
	filename                           string
	lastSeenCacheFileModifiedTimestamp int
	fileAccessor                       internal.FileAccessor
}

func NewFileCache(file string) *FileTokenCache {
	os.Create(file)
	lock, err := NewLock(file+".lock", 60, 100)
	if err != nil {
		return &FileTokenCache{}
	}
	return &FileTokenCache{
		lock:                               lock,
		filename:                           file,
		fileAccessor:                       *internal.NewFileAccessor(file),
		lastSeenCacheFileModifiedTimestamp: time.Time{},
	}
}

func (t *FileTokenCache) Replace(cache cache.Unmarshaler, key string) error {
	info, err := os.Stat(t.filename)
	if err != nil {
		return err
	}
	currentCacheFileModifiedTime := info.ModTime()
	if currentCacheFileModifiedTime != t.lastSeenCacheFileModifiedTimestamp {
		if err := t.lock.Lock(); err != nil {
			return err
		}
		defer t.lock.UnLock()
		data, err := t.fileAccessor.Read()
		if err != nil {
			return err
		}
		err = cache.Unmarshal(data)
		if err != nil {
			return err
		}
	}
}

func (t *FileTokenCache) Export(cache cache.Marshaler, key string) error {
	if err := t.lock.Lock(); err != nil {
		return err
	}
	defer t.lock.UnLock()
	data, err := cache.Marshal()
	if err != nil {
		return err
	}
	t.fileAccessor.Write(data)
	info, err := os.Stat(t.filename)
	if err != nil {
		return err
	}
	t.lastSeenCacheFileModifiedTimestamp := info.ModTime()
}

type WindowsTokenCache struct {
	lock                               CrossPlatLock
	filename                           string
	lastSeenCacheFileModifiedTimestamp int
	windowsAccessor                    internal.WindowsAccessor
}

func NewWindowsCache(file string) *WindowsTokenCache {
	os.Create(file)
	lock, err := NewLock(file+".lock", 60, 100)
	if err != nil {
		return &WindowsTokenCache
	}
	return &WindowsTokenCache{
		lock:            lock,
		filename:        file,
		windowsAccessor: *internal.NewWindowsAccessor(filepath),
		lastSeenCacheFileModifiedTimestamp: time.Time{},
	}
}

func (t *WindowsTokenCache) Replace(cache cache.Unmarshaler, key string) error {
	info, err := os.Stat(t.filename)
	if err != nil {
		return err
	}
	currentCacheFileModifiedTime := info.ModTime()
		if currentCacheFileModifiedTime != t.lastSeenCacheFileModifiedTimestamp {
		t.lock.Lock()
		defer t.lock.UnLock()
		data, err := t.windowsAccessor.Read()
		if err != nil {
			return err
		}
		err = cache.Unmarshal(data)
		if err != nil {
			return err
		}
	}
}

func (t *WindowsTokenCache) Export(cache cache.Marshaler, key string) error {
	t.lock.Lock()
	defer t.lock.UnLock()
	data, err := cache.Marshal()
	if err != nil {
		return err
	}
	t.windowsAccessor.Write(data)
	
	info, err := os.Stat(t.filename)
	if err != nil {
		return err
	}
	t.lastSeenCacheFileModifiedTimestamp := info.ModTime()
}
