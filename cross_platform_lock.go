package main

import (
	"fmt"
	"os"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/flock"
)

type CrossPlatLock struct {
	retryNumber            int
	retryDelayMilliseconds int

	lockFile *os.File

	fileLock *flock.Flock

	lockfileName string
	locked       bool
}

func NewLock(lockFileName string, retryNumber int) (CrossPlatLock, error) {
	lockfile, err := os.Create("cache_trial.json.lockfile")
	if err != nil {
		fmt.Println(err)
	}
	return CrossPlatLock{
		lockfileName: lockFileName,
		retryNumber:  retryNumber,
		lockFile:     lockfile,
		fileLock:     flock.New(lockfile.Name()),
	}, nil
}

func (c CrossPlatLock) Lock() error {

	err := c.fileLock.Lock()
	if err != nil {
		return err
	}
	if c.fileLock.Locked() {
		c.fileLock.Fh.WriteString("Hello \n")
	}
	c.locked = true
	time.Sleep(10)
	return nil
}

func (c CrossPlatLock) UnLock() error {
	if c.fileLock != nil {
		if err := c.fileLock.Unlock(); err != nil {
			fmt.Println("UnLock ", err.Error())
			// handle unlock error
		}
		c.lockFile.Close()
		var err = os.Remove(c.fileLock.Path())
		if err != nil {
			fmt.Println(err.Error())
		}
		c.locked = false
	}
	return nil
}
