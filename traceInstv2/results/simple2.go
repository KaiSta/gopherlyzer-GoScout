package main

import "../tracer"

func foo(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\simple2.go:4")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\simple2.go:4")
}

func main() {
	tracer.Start()
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	tmp1 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp1)
	go func(x chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("foo0", tmp1)

		foo(x)
	}(x)
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(x chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("foo1", tmp2)

		foo(x)
	}(x)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\simple2.go:11")
	tmp3 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\simple2.go:11", tmp3.threadId)
	tracer.Stop()
}
