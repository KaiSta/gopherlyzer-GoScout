package main

import "sync"

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	m1 := sync.Mutex{}
	m2 := sync.Mutex{}
	x := 0
	tracer.WriteAcc(&x, "ex2.go:8", myTIDCache)
	tracer.PreLock(&m1, "ex2.go:10", myTIDCache)
	m1.Lock()
	tracer.WriteAcc(&x, "ex2.go:11", myTIDCache)
	x++
	tracer.PostLock(&m1, "ex2.go:12", myTIDCache)
	m1.Unlock()
	tracer.PreLock(&m2, "ex2.go:13", myTIDCache)
	m2.Lock()
	tracer.WriteAcc(&x, "ex2.go:14", myTIDCache)
	x++
	tracer.PostLock(&m2, "ex2.go:15", myTIDCache)
	m2.Unlock()
}
