package main

import (
	"sync"
)

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	m := sync.Mutex{}
	var x int
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.WriteAcc(&x, "ex1.go:13", myTIDCache)
		x++
		tracer.PreLock(&m, "ex1.go:14", myTIDCache)
		m.Lock()
		tracer.PostLock(&m, "ex1.go:15", myTIDCache)
		m.Unlock()
		tracer.PreSend(c, "ex1.go:16", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex1.go:16", myTIDCache)
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&m, "ex1.go:19", myTIDCache)
		m.Lock()
		tracer.PostLock(&m, "ex1.go:20", myTIDCache)
		m.Unlock()
		tracer.WriteAcc(&x, "ex1.go:21", myTIDCache)
		x++
		tracer.PreSend(c, "ex1.go:22", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex1.go:22", myTIDCache)
	}()
	tracer.PreRcv(c, "ex1.go:25", myTIDCache)
	tmp3 := <-c
	tracer.PostRcv(c, "ex1.go:25", tmp3.threadId, myTIDCache)
	tracer.PreRcv(c, "ex1.go:26", myTIDCache)
	tmp4 := <-c
	tracer.PostRcv(c, "ex1.go:26", tmp4.threadId, myTIDCache)
}
