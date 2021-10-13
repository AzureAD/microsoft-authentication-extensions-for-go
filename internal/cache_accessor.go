package internal

import (
	"io/ioutil"
	"os"
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
		return nil, err
	}
	defer file.Close()
	data, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *FileAccessor) Write(data []byte) error {
	err := ioutil.WriteFile(f.cacheFilePath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileAccessor) WriteAtomic(data []byte) {
	// Not implemented yet
}

func (f *FileAccessor) Delete() {

}
