package lock

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/flock"
)

type Lock struct {
	retries    int
	retryDelay time.Duration

	lockfileName string

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
	l.fLock = flock.New(lockFileName)
	l.lockfileName = lockFileName
	return l, nil
}

func (l *Lock) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for tryCount := 0; tryCount < l.retries; tryCount++ {
		locked, err := l.fLock.TryLock()
		if err != nil || !locked {
			time.Sleep(l.retryDelay * time.Millisecond)
			continue
		} else if locked {
			// commenting this till we have flock merged in and ready to use with changes
			// otherwise it would not compile
			//l.fLock.Fh.WriteString(fmt.Sprintf("{%d} {%s}", os.Getpid(), os.Args[0]))
			//l.fLock.Fh.Sync()
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
		err := os.Remove(l.fLock.Path())
		if err != nil {
			errVal := err.(*fs.PathError)
			if errVal != fs.ErrNotExist || errVal != fs.ErrPermission {
				return err
			}
		}
	}
	return nil
}
