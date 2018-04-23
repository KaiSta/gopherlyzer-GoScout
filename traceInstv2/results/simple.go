package main

import "../tracer"

func foo(x chan struct {
	threadId uint64
	value    int
}, y int) {
	tracer.ReadAcc(&y, "tests\\simple.go:4", tracer.GetGID())
	tracer.PreSend(x, "tests\\simple.go:4", tracer.GetGID())
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), y}
	tracer.PostSend(x, "tests\\simple.go:4", tracer.GetGID())
	tracer.PreSend(x, "tests\\simple.go:5", tracer.GetGID())
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.PostSend(x, "tests\\simple.go:5", tracer.GetGID())
	tracer.PreSend(x, "tests\\simple.go:6", tracer.GetGID())
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 2}
	tracer.PostSend(x, "tests\\simple.go:6", tracer.GetGID())
	tracer.PreSend(x, "tests\\simple.go:7", tracer.GetGID())
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 3}
	tracer.PostSend(x, "tests\\simple.go:7", tracer.GetGID())
}

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())
	go func(x chan struct {
		threadId uint64
		value    int
	}, y int) {
		tracer.RegisterThread("foo0")
		tracer.Wait(tmp1, tracer.GetGID())

		foo(x, 42)
	}(x, 42)
	tracer.PreRcv(x, "tests\\simple.go:13", tracer.GetGID())
	tmp2 := <-x
	tracer.PostRcv(x, "tests\\simple.go:13", tmp2.threadId, tracer.GetGID())
	tracer.PreRcv(x, "tests\\simple.go:14", tracer.GetGID())
	tmp3 := <-x
	tracer.PostRcv(x, "tests\\simple.go:14", tmp3.threadId, tracer.GetGID())
	tracer.PreRcv(x, "tests\\simple.go:14", tracer.GetGID())
	tmp4 := <-x
	tracer.PostRcv(x, "tests\\simple.go:14", tmp4.threadId, tracer.GetGID())
	z := (tmp3.value) + tmp4.value
	tracer.WriteAcc(&z, "tests\\simple.go:14", tracer.GetGID())
	tracer.Stop()
}
