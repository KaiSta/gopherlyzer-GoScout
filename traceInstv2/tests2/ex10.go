package main

import "sync"

func main() {
	m := sync.Mutex{}
	c := make(chan int)
	x := 0

	go func() {
		m.Lock()   //L1
		x++        //L2
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
