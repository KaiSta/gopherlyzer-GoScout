package main

import "../tracer"

var z int

func foo(x chan struct {
	threadId uint64
	value    int
}, y chan struct {
	threadId uint64
	value    int
}) {
	tracer.PreSend(x, "tests\\async.go:6", tracer.GetGID())
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.PostSend(x, "tests\\async.go:6", tracer.GetGID())
	tracer.WriteAcc(&z, "tests\\async.go:7", tracer.GetGID())
	z++
	tracer.PreRcv(x, "tests\\async.go:8", tracer.GetGID())
	tmp1 := <-x
	tracer.PostRcv(x, "tests\\async.go:8", tmp1.threadId, tracer.GetGID())
	tracer.PreSend(y, "tests\\async.go:9", tracer.GetGID())
	y <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.PostSend(y, "tests\\async.go:9", tracer.GetGID())
}

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	x := make(chan struct {
		threadId uint64
		value    int
	}, 1)
	tracer.RegisterChan(x, cap(x))
	y := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(y, cap(y))
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, tracer.GetGID())
	go func(x chan struct {
		threadId uint64
		value    int
	},

		y chan struct {
			threadId uint64
			value    int
		}) {
		tracer.RegisterThread("foo0")
		tracer.Wait(tmp2, tracer.GetGID())

		foo(x, y)
	}(x, y)
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, tracer.GetGID())
	go func(x chan struct {
		threadId uint64
		value    int
	},

		y chan struct {
			threadId uint64
			value    int
		}) {
		tracer.RegisterThread("foo1")
		tracer.Wait(tmp3, tracer.GetGID())

		foo(x, y)
	}(x, y)
	tracer.PreRcv(y, "tests\\async.go:18", tracer.GetGID())
	tmp4 := <-y
	tracer.PostRcv(y, "tests\\async.go:18", tmp4.threadId, tracer.GetGID())
	tracer.PreRcv(y, "tests\\async.go:19", tracer.GetGID())
	tmp5 := <-y
	tracer.PostRcv(y, "tests\\async.go:19", tmp5.threadId, tracer.GetGID())
	tracer.Stop()
}
