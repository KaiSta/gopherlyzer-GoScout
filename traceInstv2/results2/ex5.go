package main

import "sync"

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	m := sync.Mutex{}
	tracer.PreLock(&m, "ex5.go:7", myTIDCache)
	m.Lock()
	x := 0
	tracer.WriteAcc(&x, "ex5.go:8", myTIDCache)
	tracer.PostLock(&m, "ex5.go:9", myTIDCache)
	m.Unlock()
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.WriteAcc(&x, "ex5.go:11", myTIDCache)
		x++
	}()

}
