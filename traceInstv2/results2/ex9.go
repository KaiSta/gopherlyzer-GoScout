package main

import "sync"

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	m := sync.Mutex{}
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	x := 0
	tracer.WriteAcc(&x, "ex9.go:8", myTIDCache)
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&m, "ex9.go:11", myTIDCache)
		m.Lock()
		tracer.WriteAcc(&x, "ex9.go:12", myTIDCache)
		x++
		tracer.PostLock(&m, "ex9.go:13", myTIDCache)
		m.Unlock()
		tracer.PreSend(c, "ex9.go:14", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex9.go:14", myTIDCache)
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&m, "ex9.go:17", myTIDCache)
		m.Lock()
		tracer.WriteAcc(&x, "ex9.go:18", myTIDCache)
		x++
		tracer.PostLock(&m, "ex9.go:19", myTIDCache)
		m.Unlock()
		tracer.PreSend(c, "ex9.go:20", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex9.go:20", myTIDCache)
	}()
	tracer.PreRcv(c, "ex9.go:22", myTIDCache)
	tmp3 := <-c
	tracer.PostRcv(c, "ex9.go:22", tmp3.threadId, myTIDCache)
	tracer.PreRcv(c, "ex9.go:23", myTIDCache)
	tmp4 := <-c
	tracer.PostRcv(c, "ex9.go:23", tmp4.threadId, myTIDCache)
}
