package lock

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/flock"
)

type Lock struct {
	retries    int
	retryDelay time.Duration

	lockFile *os.File

	fLock *flock.Flock
	mu    sync.Mutex
}

type Option func(l *Lock)

func WithRetries(n int) Option {
	return func(l *Lock) {
		l.retries = n
	}
}
func WithRetryDelay(t time.Duration) Option {
	return func(l *Lock) {
		l.retryDelay = t
	}
}

func New(lockFileName string, options ...Option) (*Lock, error) {
	l := &Lock{}
	for _, o := range options {
		o(l)
	}
	lockfile, err := os.Create(lockFileName)
	if err != nil {
		return &Lock{}, err
	}
	l.fLock = flock.New(lockfile.Name())
	return l, nil
}

func (l *Lock) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for tryCount := 0; tryCount < l.retries; tryCount++ {
		err := l.fLock.Lock()
		if err != nil {
			time.Sleep(l.retryDelay * time.Millisecond)
			continue
		} else {
			if l.fLock.Locked() {
				l.fLock.Fh.WriteString(fmt.Sprintf("{%d} {%s}", os.Getpid(), os.Args[0]))
			}
			return nil
		}
	}
	return errors.New("failed to acquire lock")
}

func (l *Lock) UnLock() error {
	if l.fLock != nil {
		if err := l.fLock.Unlock(); err != nil {
			return err
		}
		l.lockFile.Close()
		if err := os.Remove(l.fLock.Path()); err != nil {
			return err
		}
	}
	return nil
}
