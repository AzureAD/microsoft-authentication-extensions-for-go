package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/billgraziano/dpapi"
)

type cacheAccessor interface {
	Read() ([]byte, error)
	Write(data []byte)
	Delete()
}

type FileAccessor struct {
	cacheFilePath string
}

func NewFileAccessor(cacheFilePath string) *FileAccessor {
	return &FileAccessor{cacheFilePath: cacheFilePath}
}

func (f *FileAccessor) Read() ([]byte, error) {
	var data []byte
	file, err := os.Open(f.cacheFilePath)
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

func (f *FileAccessor) Write(data []byte) {
	if runtime.GOOS == "windows" {
		data, err := dpapi.EncryptBytes(data)
		if err != nil {
			fmt.Println("Error from Encrypt")
		}
		err = ioutil.WriteFile(f.cacheFilePath, data, 0600)
		if err != nil {
			log.Println(err)
		}
	} else {
		f.WriteAtomic(data)
	}

}

func (f *FileAccessor) WriteAtomic(data []byte) {
	// Not implemented yet
	return
}

func (f *FileAccessor) Delete() {

}
