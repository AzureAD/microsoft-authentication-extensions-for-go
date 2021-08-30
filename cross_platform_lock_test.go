package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func doSomething(i int) {
	cacheAccessor := NewFileCache("lock.lock")
	cacheAccessor.lock.Lock()
	defer cacheAccessor.lock.UnLock()
	file, err := os.OpenFile("lockintervals.txt", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	file.WriteString(fmt.Sprintf("< %d \n", i))
	time.Sleep(1 * time.Second)
	file.WriteString(fmt.Sprintf("> %d \n", i))
}
func validateResult() int {
	count := 0
	var prevProc string = ""
	var tag string
	var proc string
	data, err := os.ReadFile("lockintervals.txt")
	if err != nil {
		fmt.Println(err)
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
	}
	return count
}

func TestCrossPlatLock(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(30)
	for i := 0; i < 30; i++ {
		go func(i int) {
			defer wg.Done()
			doSomething(i)
		}(i)
	}
	wg.Wait()
	n := validateResult()
	fmt.Println(n)
	if n > 60 {
		fmt.Println("Should not observe starvation")
	}
}
