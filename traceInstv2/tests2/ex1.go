package main

import (
	"sync"
)

func main() {
	c := make(chan int)
	m := sync.Mutex{}
	var x int

	go func() {
		x++        //L1
		m.Lock()   //L2
		m.Unlock() //L3
		c <- 1
	}()
	go func() {
		m.Lock()   //L4
		m.Unlock() //L5
		x++        //L6
		c <- 1
	}()

	<-c
	<-c
}
