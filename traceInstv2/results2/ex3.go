package main

import "sync"

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	m := sync.Mutex{}
	tracer.PreLock(&m, "ex3.go:7", myTIDCache)
	m.Lock()
	x := 0
	tracer.WriteAcc(&x, "ex3.go:8", myTIDCache)
	tracer.PostLock(&m, "ex3.go:9", myTIDCache)
	m.Unlock()
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.ReadAcc(&x, "ex3.go:11", myTIDCache)
		print(x)
	}()

}
