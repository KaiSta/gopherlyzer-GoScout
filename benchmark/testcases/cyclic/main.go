package main

import (
	"sync"
	"time"
)

func main() {
	a := sync.Mutex{}
	b := sync.Mutex{}

	go func() {
		a.Lock()
		time.Sleep(1 * time.Second)
		b.Lock()
		b.Unlock()
		a.Unlock()
	}()

	b.Lock()
	time.Sleep(1 * time.Second)
	a.Lock()
	a.Unlock()
	b.Unlock()
}
