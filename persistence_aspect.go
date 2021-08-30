package main

import (
	"log"

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
	lock, err := NewLock(file, 600)
	if err != nil {
		log.Print(err)
	}
	return &FileTokenCache{
		lock:         lock,
		fileAccessor: *internal.NewFileAccessor("cache_trial.json"),
	}
}

func (t *FileTokenCache) Replace(cache cache.Unmarshaler, key string) {
	t.lock.Lock()

	data, err := t.fileAccessor.Read()

	if err != nil {
		log.Println(err)
	}
	err = cache.Unmarshal(data)
	if err != nil {
		log.Println(err)
	}
	t.lock.UnLock()
}

func (t *FileTokenCache) Export(cache cache.Marshaler, key string) {
	t.lock.Lock()
	data, err := cache.Marshal()
	if err != nil {
		log.Println(err)
	}
	t.fileAccessor.Write(data)
	t.lock.UnLock()
}
