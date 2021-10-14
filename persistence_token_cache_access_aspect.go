//go:build windows
// +build windows

package main

import (
	"log"
	"os"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/internal"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type FileTokenCache struct {
	lock                               CrossPlatLock
	filename                           string
	lastSeenCacheFileModifiedTimestamp time.Time
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

func (t *FileTokenCache) Replace(cache cache.Unmarshaler, key string) {
	info, err := os.Stat(t.filename)
	if err != nil {
		log.Println(err)
	}
	currentCacheFileModifiedTime := info.ModTime()
	if currentCacheFileModifiedTime != t.lastSeenCacheFileModifiedTimestamp {
		if err := t.lock.Lock(); err != nil {
			log.Println(err)
		}
		defer t.lock.UnLock()
		data, err := t.fileAccessor.Read()
		if err != nil {
			log.Println(err)
		}
		err = cache.Unmarshal(data)
		if err != nil {
			log.Println(err)
		}
	}
}

func (t *FileTokenCache) Export(cache cache.Marshaler, key string) {
	if err := t.lock.Lock(); err != nil {
		log.Println(err)
	}
	defer t.lock.UnLock()
	data, err := cache.Marshal()
	if err != nil {
		log.Println(err)
	}
	t.fileAccessor.Write(data)
	info, err := os.Stat(t.filename)
	if err != nil {
		log.Println(err)
	}
	t.lastSeenCacheFileModifiedTimestamp = info.ModTime()
}

type WindowsTokenCache struct {
	lock                               CrossPlatLock
	filename                           string
	lastSeenCacheFileModifiedTimestamp time.Time
	windowsAccessor                    internal.WindowsAccessor
}

func NewWindowsCache(file string) *WindowsTokenCache {
	os.Create(file)
	lock, err := NewLock(file+".lock", 60, 100)
	if err != nil {
		return &WindowsTokenCache{}
	}
	return &WindowsTokenCache{
		lock:                               lock,
		filename:                           file,
		windowsAccessor:                    *internal.NewWindowsAccessor(file),
		lastSeenCacheFileModifiedTimestamp: time.Time{},
	}
}

func (t *WindowsTokenCache) Replace(cache cache.Unmarshaler, key string) {
	info, err := os.Stat(t.filename)
	if err != nil {
		log.Println(err)
	}
	currentCacheFileModifiedTime := info.ModTime()
	if currentCacheFileModifiedTime != t.lastSeenCacheFileModifiedTimestamp {
		t.lock.Lock()
		defer t.lock.UnLock()
		data, err := t.windowsAccessor.Read()
		if err != nil {
			log.Println(err)
		}
		err = cache.Unmarshal(data)
		if err != nil {
			log.Println(err)
		}
	}
}

func (t *WindowsTokenCache) Export(cache cache.Marshaler, key string) {
	t.lock.Lock()
	defer t.lock.UnLock()
	data, err := cache.Marshal()
	if err != nil {
		log.Println(err)
	}
	t.windowsAccessor.Write(data)

	info, err := os.Stat(t.filename)
	if err != nil {
		log.Println(err)
	}
	t.lastSeenCacheFileModifiedTimestamp = info.ModTime()
}
