package main

import "sync"

func main() { //T1
	m := sync.Mutex{}
	m.Lock()    //L1
	x := 0      //L2
	m.Unlock()  //L3
	go func() { //T2
		x++ //L4 print(x) in the
		//previous example.
	}()
}
