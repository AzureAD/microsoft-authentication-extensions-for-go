package lock

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/internal"
)

const cacheFile = "cache.txt"

func TestLocking(t *testing.T) {

	tests := []struct {
		desc          string
		concurrency   int
		sleepInterval time.Duration
		cacheFile     string
	}{
		{"Normal", 4, 50 * time.Millisecond, "cache_normal"},
		{"High", 40, 50 * time.Millisecond, "cache_high"},
	}

	for _, test := range tests {
		tmpfile, err := ioutil.TempFile("", test.cacheFile)
		defer os.Remove(tmpfile.Name())
		if err != nil {
			t.Fatalf("TestLocking(%s): Could not create cache file", test.desc)
		}
		err = spin(test.concurrency, time.Duration(test.sleepInterval), tmpfile.Name())
		if err != nil {
			t.Fatalf("TestLocking(%s): %s", test.desc, err)
		}
	}

}
func acquire(threadNo int, sleepInterval time.Duration, cacheFile string) {
	cacheAccessor := internal.NewFileAccessor(cacheFile)
	lockfileName := cacheFile + ".lockfile"
	l, err := New(lockfileName, WithRetries(60), WithRetryDelay(100))
	if err := l.Lock(); err != nil {
		log.Println("Couldn't acquire lock", err.Error())
		return
	}
	defer l.UnLock()
	data, err := cacheAccessor.Read()
	if err != nil {
		log.Println(err)
	}
	var buffer bytes.Buffer
	buffer.Write(data)
	buffer.WriteString(fmt.Sprintf("< %d \n", threadNo))
	time.Sleep(sleepInterval)
	buffer.WriteString(fmt.Sprintf("> %d \n", threadNo))
	cacheAccessor.Write(buffer.Bytes())
}

func spin(concurrency int, sleepInterval time.Duration, cacheFile string) error {
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer wg.Done()
			acquire(i, sleepInterval, cacheFile)
		}(i)
	}
	wg.Wait()
	i, err := validate(cacheFile)
	if err != nil {
		return err
	}
	if i != concurrency*2 {
		return fmt.Errorf("should have seen %d line entries, found %d", concurrency*2, i)
	}
	return nil
}

func validate(cacheFile string) (int, error) {
	var (
		count               int
		prevProc, tag, proc string
	)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		log.Println(err)
	}
	temp := strings.Split(string(data), "\n")

	for _, s := range temp {
		if strings.TrimSpace(s) == "" {
			continue
		}
		count += 1
		split := strings.Split(s, " ")
		tag = split[0]
		proc = split[1]
		if prevProc == "" {
			if tag != "<" {
				return 0, errors.New("opening bracket not found")
			}
			prevProc = proc
			continue
		}
		if proc != prevProc {
			return 0, errors.New("process overlap found")
		}
		if tag != ">" {
			return 0, errors.New("process overlap found")
		}
		prevProc = ""
	}
	return count, nil
}
