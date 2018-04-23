package main

import "fmt"
import "../tracer"

func foo(x chan struct {
	threadId uint64
	value    int
}, y int) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:6")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), y}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:6")
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
	}, int) {
		tracer.RegisterThread("foo0", tmp1)

		foo(x, 1)
	}(x, 1)
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(x chan struct {
		threadId uint64
		value    int
	}, int) {
		tracer.RegisterThread("foo1", tmp2)

		foo(x, 2)
	}(x, 2)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:14")
	tmp3 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:14", tmp3.threadId)
	y := tmp3.value
	fmt.Println(y)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:17")
	tmp4 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:17", tmp4.threadId)
	y = tmp4.value
	fmt.Println(y)
	tracer.ClosePrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:20")
	close(x)
	tracer.CloseCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:20")
	close(x)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:22")
	tmp5 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:22", tmp5.threadId)
	tmp6 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp6)

	go func() {
		tracer.RegisterThread("fun2", tmp6)
		tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:25")
		tmp7 := <-x
		tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:25", tmp7.threadId)
	}()
	tracer.Stop()
}
