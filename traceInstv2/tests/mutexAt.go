package main

import (
	"sync"
)

var z int

func foo(m *sync.Mutex, y chan int) {
	m.Lock()
	z++
	m.Unlock()

	y <- 1
}

func main() {
	m := sync.Mutex{}

	y := make(chan int)

	go foo(&m, y)
	go foo(&m, y)

	<-y
	<-y

}
