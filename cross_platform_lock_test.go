package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AzureAD/microsoft-authentication-extensions-for-go/internal"
)

func spinThreads(noOfThreads int, sleepInterval time.Duration) int {
	cacheFile := "cache.txt"
	var wg sync.WaitGroup
	wg.Add(noOfThreads)
	for i := 0; i < noOfThreads; i++ {
		go func(i int) {
			defer wg.Done()
			acquireLockAndWriteToCache(i, sleepInterval, cacheFile)
		}(i)
	}
	wg.Wait()
	return validateResult(cacheFile)
}

func acquireLockAndWriteToCache(threadNo int, sleepInterval time.Duration, cacheFile string) {
	cacheAccessor := internal.NewFileAccessor(cacheFile)
	lockfileName := cacheFile + ".lockfile"
	lock, err := NewLock(lockfileName, 60, 100)
	if err := lock.Lock(); err != nil {
		log.Println("Couldn't acquire lock", err.Error())
		return
	}
	defer lock.UnLock()
	data, err := cacheAccessor.Read()
	if err != nil {
		log.Println(err)
	}
	var buffer bytes.Buffer
	buffer.Write(data)
	buffer.WriteString(fmt.Sprintf("< %d \n", threadNo))
	time.Sleep(sleepInterval * time.Millisecond)
	buffer.WriteString(fmt.Sprintf("> %d \n", threadNo))
	cacheAccessor.Write(buffer.Bytes())
}

func validateResult(cacheFile string) int {
	count := 0
	var prevProc string = ""
	var tag string
	var proc string
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		log.Println(err)
	}
	dat := string(data)
	temp := strings.Split(dat, "\n")
	for _, ele := range temp {
		if ele != "" {
			count += 1
			split := strings.Split(ele, " ")
			tag = split[0]
			proc = split[1]
			if prevProc != "" {
				if proc != prevProc {
					fmt.Println("Process overlap found")
				}
				if tag != ">" {
					fmt.Println("Process overlap found 1")
				}
				prevProc = ""

			} else {
				if tag != "<" {
					fmt.Println("Opening bracket not found")
				}
				prevProc = proc
			}
		}
		if err := os.Remove(cacheFile); err != nil {
			log.Println("Failed to remove cache file", err)
		}
	}
	return count
}
func TestForNormalWorkload(t *testing.T) {
	noOfThreads := 4
	sleepInterval := 100
	n := spinThreads(noOfThreads, time.Duration(sleepInterval))
	if n != 4*2 {
		t.Fatalf("Should not observe starvation")
	}
}

func TestForHighWorkload(t *testing.T) {
	noOfThreads := 80
	sleepInterval := 100
	n := spinThreads(noOfThreads, time.Duration(sleepInterval))
	if n > 80*2 {
		t.Fatalf("Starvation or not, we should not observe garbled payload")
	}
}
