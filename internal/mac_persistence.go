//go:build darwin
// +build darwin

package internal

import (
	"time"

	"github.com/keybase/go-keychain"
)

type KeyChainAccessor struct {
	cacheFilePath string
	serviceName   string
	accountName   string
}

func NewKeyChainAccessor(cacheFilePath string, serviceName string, accountName string) *KeyChainAccessor {
	return &KeyChainAccessor{cacheFilePath: cacheFilePath, serviceName: serviceName, accountName: accountName}
}

func (k *KeyChainAccessor) Read() ([]byte, error) {
	var data []byte
	data, err := keychain.GetGenericPassword(k.serviceName, k.accountName, "", "")
	if err == nil && data == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (k *KeyChainAccessor) WriteNoRetry(data []byte) error {
	pass, err := keychain.GetGenericPassword(k.serviceName, k.accountName, "", "")
	if pass == nil && err == nil {
		// Data and error nil correspond to ITEM_NOT_FOUND
		item := keychain.NewGenericPassword(k.serviceName, k.accountName, "AddingNewEntry", data, "")
		if err = keychain.AddItem(item); err != nil {
			return err
		}

	}
	if err == nil {
		item := keychain.NewItem()
		item.SetSecClass(keychain.SecClassGenericPassword)
		item.SetService(k.serviceName)
		item.SetAccount(k.accountName)

		updateItem := keychain.NewItem()
		updateItem.SetSecClass(keychain.SecClassGenericPassword)
		updateItem.SetService(k.serviceName)
		updateItem.SetAccount(k.accountName)
		updateItem.SetLabel("UpdatingExistingEntry")
		updateItem.SetData(data)
		err = keychain.UpdateItem(item, updateItem)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

func (k *KeyChainAccessor) Write(data []byte) error {
	noOfRetries := 3
	var retryDelay time.Duration = 10
	var err error
	for i := 0; i < noOfRetries; i++ {
		err := k.WriteNoRetry(data)
		if err == nil {
			// Update the last modified time of file
			return nil
		}
		time.Sleep(retryDelay * time.Millisecond)
	}
	return err
}

func (w *KeyChainAccessor) Delete() {
	//Not Implemented
}
