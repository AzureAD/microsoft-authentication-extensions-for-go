//go:build darwin
// +build darwin

package main

import (
	"log"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/internal"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type MacTokenCache struct {
	lock                               CrossPlatLock
	filename                           string
	lastSeenCacheFileModifiedTimestamp int
	keyChainAccessor                   internal.KeyChainAccessor
}

func NewMacTokenCache(file string, serviceName string, accountName string) *MacTokenCache {
	lock, err := NewLock(file, 60, 100)
	if err != nil {
		log.Print(err)
	}
	return &MacTokenCache{
		lock:             lock,
		keyChainAccessor: *internal.NewKeyChainAccessor("cache_trial.json", serviceName, accountName),
	}
}

func (t *MacTokenCache) Replace(cache cache.Unmarshaler, key string) {
	t.lock.Lock()

	data, err := t.keyChainAccessor.Read()

	if err != nil {
		log.Println(err)
	}
	err = cache.Unmarshal(data)
	if err != nil {
		log.Println(err)
	}
	t.lock.UnLock()
}

func (t *MacTokenCache) Export(cache cache.Marshaler, key string) {
	t.lock.Lock()
	data, err := cache.Marshal()
	if err != nil {
		log.Println(err)
	}
	t.keyChainAccessor.Write(data)
	t.lock.UnLock()
}
