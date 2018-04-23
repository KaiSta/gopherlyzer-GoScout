package main

import "sync"

var x int

func Reader(m *sync.Mutex) {
	myTIDCache := tracer.GetGID()
	for {
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&m, "ex4.go:9", myTIDCache)
		m.Lock()
		tracer.ReadAcc(&x, "ex4.go:10", myTIDCache)
		print(x)
		tracer.PostLock(&m, "ex4.go:11", myTIDCache)
		m.Unlock()
	}
}
func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	a := sync.Mutex{}
	b := sync.Mutex{}
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func(m *sync.Mutex) { tracer.RegisterThread("Reader0"); tracer.Wait(tmp1, tracer.GetGID()); Reader(m) }(&a)
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)
	go func(m *sync.Mutex) { tracer.RegisterThread("Reader1"); tracer.Wait(tmp2, tracer.GetGID()); Reader(m) }(&b)

	for i := 0; i < 10; i++ {
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&a, "ex4.go:21", myTIDCache)
		a.Lock()
		tracer.PreLock(&b, "ex4.go:22", myTIDCache)
		b.Lock()
		tracer.WriteAcc(&x, "ex4.go:23", myTIDCache)
		x++
		tracer.PostLock(&b, "ex4.go:24", myTIDCache)
		b.Unlock()
		tracer.PostLock(&a, "ex4.go:25", myTIDCache)
		a.Unlock()
	}
}
