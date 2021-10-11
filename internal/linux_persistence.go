//go:build gnome_keyring
// +build gnome_keyring

package internal

import (
	"time"

	"github.com/keybase/go-keychain"
)

type KeyRingAccessor struct {
	cacheFilePath string

	keyringCollection  string
	keyringSchemaName  string
	keyringSecretLabel string

	attributeKey1   string
	attributeValue1 string

	attributeKey2   string
	attributeValue2 string
	gnomeKeyRing    GnomeKeyRing
}


func NewKeyRingAccessor(cacheFilePath string,
	keyringCollection string, keyringSchemaName string,
	keyringSecretLabel string,
	attributeKey1 string, attributeValue1 string,
	attributeKey2 string, attributeValue2 string) *KeyRingAccessor {
	gnomeKeyRing := initializeProvider()
	return &KeyRingAccessor{cacheFilePath: cacheFilePath, keyringCollection: keyringCollection, keyringSchemaName: keyringSchemaName, keyringSecretLabel: keyringSecretLabel,
		attributeKey1: attributeKey1, attributeValue1: attributeValue1, attributeKey2: attributeKey2, attributeValue2: attributeValue2, gnomeKeyRing = New()}
}

func (k *KeyRingAccessor) Read() ([]byte, error) {
	var data []byte
	data, err := k.gnomeKeyRing.Get(k.attributeKey1, k.attributeValue1, k.attributeKey2, k.attributeValue2)
	if err == nil && data == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (k *KeyRingAccessor) Write(data []byte) error {
	err := k.gnomeKeyRing.Set(k.keyringSecretLabel, data, k.attributeKey1, k.attributeValue1, k.attributeKey2, k.attributeValue2)
	if err != nil {
		return nil, err
	}
}

func (w *KeyRingAccessor) Delete() {
	//Not Implemented
}
