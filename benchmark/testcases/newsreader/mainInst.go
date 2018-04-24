package main

import (
	"../../tracer"
)

func sel(x, y chan struct {
	threadId uint64
	value    bool
}) {
	myTIDCache := tracer.GetGID()
	z := make(chan struct {
		threadId uint64
		value    bool
	})
	tracer.RegisterChan(z, cap(z))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreRcv(x, "main.go:6", myTIDCache)
		tmp2 := <-x
		tracer.PostRcv(x, "main.go:6", tmp2.threadId, myTIDCache)
		tmp := tmp2.value
		tracer.PreSend(z, "main.go:7", myTIDCache)
		z <- struct {
			threadId uint64
			value    bool
		}{myTIDCache, tmp}
		tracer.PostSend(z, "main.go:7", myTIDCache)
	}()
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp3, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreRcv(y, "main.go:10", myTIDCache)
		tmp4 := <-y
		tracer.PostRcv(y, "main.go:10", tmp4.threadId, myTIDCache)
		tmp := tmp4.value
		tracer.PreSend(z, "main.go:11", myTIDCache)
		z <- struct {
			threadId uint64
			value    bool
		}{myTIDCache, tmp}
		tracer.PostSend(z, "main.go:11", myTIDCache)
	}()
	tracer.PreRcv(z, "main.go:13", myTIDCache)
	tmp5 := <-z
	tracer.PostRcv(z, "main.go:13", tmp5.threadId, myTIDCache)
}

func main() {
	tracer.Start()
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	x := make(chan struct {
		threadId uint64
		value    bool
	})
	tracer.RegisterChan(x, cap(x))
	y := make(chan struct {
		threadId uint64
		value    bool
	})
	tracer.RegisterChan(y, cap(y))
	tmp6 := tracer.GetWaitSigID()
	tracer.Signal(tmp6, myTIDCache)
	go func() {
		tracer.RegisterThread("fun2")
		tracer.Wait(tmp6, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(x, "main.go:20", myTIDCache)
		x <- struct {
			threadId uint64
			value    bool
		}{myTIDCache, true}
		tracer.PostSend(x, "main.go:20", myTIDCache)
	}()
	tmp7 := tracer.GetWaitSigID()
	tracer.Signal(tmp7, myTIDCache)

	go func() {
		tracer.RegisterThread("fun3")
		tracer.Wait(tmp7, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(y, "main.go:23", myTIDCache)
		y <- struct {
			threadId uint64
			value    bool
		}{myTIDCache, false}
		tracer.PostSend(y, "main.go:23", myTIDCache)
	}()

	sel(x, y)
	sel(x, y)
	tracer.Stop()
}
