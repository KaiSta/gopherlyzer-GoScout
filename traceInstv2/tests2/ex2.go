package main

import "sync"

func main() {
	m1 := sync.Mutex{}
	m2 := sync.Mutex{}
	x := 0

	m1.Lock()   //L1
	x++         //L2
	m1.Unlock() //L3
	m2.Lock()   //L4
	x++         //L5
	m2.Unlock() //L6
}
