package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/flock"
)

type CrossPlatLock struct {
	retryNumber            int
	retryDelayMilliseconds time.Duration

	lockFile *os.File

	fileLock *flock.Flock

	lockfileName string
	locked       bool
}

func NewLock(lockFileName string, retryNumber int, retryDelayMilliseconds time.Duration) (CrossPlatLock, error) {
	lockfile, err := os.Create(lockFileName)
	if err != nil {
		return CrossPlatLock{}, err
	}
	return CrossPlatLock{
		lockfileName: lockFileName,
		retryNumber:  retryNumber,
		lockFile:     lockfile,
		fileLock:     flock.New(lockfile.Name()),
	}, nil
}

func (c CrossPlatLock) Lock() error {
	for tryCount := 0; tryCount < c.retryNumber; tryCount++ {
		err := c.fileLock.Lock()
		if err != nil {
			time.Sleep(c.retryDelayMilliseconds * time.Millisecond)
			continue
		} else {
			if c.fileLock.Locked() {
				c.fileLock.Fh.WriteString(fmt.Sprintf("{%d} {%s}", os.Getpid(), os.Args[0]))
			}
			c.locked = true
			return nil
		}
	}
	return errors.New("Failed to acquire lock")
}

func (c CrossPlatLock) UnLock() error {
	if c.fileLock != nil {
		if err := c.fileLock.Unlock(); err != nil {
			return err
		}
		c.lockFile.Close()
		if err := os.Remove(c.fileLock.Path()); err != nil {
			return err
		}
		c.locked = false
	}
	return nil
}
