package main

import "sync"

var x int

func Reader(m *sync.Mutex) {
	for {
		m.Lock()   //L1
		print(x)   //L2
		m.Unlock() //L3
	}
}
func main() {
	a := sync.Mutex{}
	b := sync.Mutex{}
	go Reader(&a)
	go Reader(&b)
	//writer
	for i := 0; i < 10; i++ {
		a.Lock()   //L4
		b.Lock()   //L5
		x++        //L6
		b.Unlock() //L7
		a.Unlock() //L8
	}
}
