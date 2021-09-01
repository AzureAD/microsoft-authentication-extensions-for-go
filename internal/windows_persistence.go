// +build windows

package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/billgraziano/dpapi"
)

type WindowsAccessor struct {
	cacheFilePath string
}

func NewWindowsAccessor(cacheFilePath string) *WindowsAccessor {
	return &WindowsAccessor{cacheFilePath: cacheFilePath}
}

func (w *WindowsAccessor) Read() ([]byte, error) {
	var data []byte
	file, err := os.Open(w.cacheFilePath)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	data, err = ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
	}
	if data != nil && len(data) != 0 && runtime.GOOS == "windows" {
		data, err = dpapi.DecryptBytes(data)
		if err != nil {
			fmt.Println("err from Decrypt: ", err)
		}
	}
	return data, nil
}

func (w *WindowsAccessor) Write(data []byte) {
	if runtime.GOOS == "windows" {
		data, err := dpapi.EncryptBytes(data)
		if err != nil {
			fmt.Println("Error from Encrypt")
		}
		err = ioutil.WriteFile(w.cacheFilePath, data, 0600)
		if err != nil {
			log.Println(err)
		}
	} else {
		w.WriteAtomic(data)
	}

}

func (w *WindowsAccessor) WriteAtomic(data []byte) {
	// Not implemented yet
	return
}

func (w *WindowsAccessor) Delete() {

}
