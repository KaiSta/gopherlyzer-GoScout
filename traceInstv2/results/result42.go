package main

import "fmt"

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
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	go func(x chan struct {
		threadId uint64
		value    int
	}, int) {
		tracer.RegisterThread("foo0")

		foo(x, 1)
	}(x, 1)
	go func(x chan struct {
		threadId uint64
		value    int
	}, int) {
		tracer.RegisterThread("foo1")

		foo(x, 2)
	}(x, 2)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:14")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:14", tmp1.threadId)
	y := tmp1.value
	fmt.Println(y)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:17")
	tmp2 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:17", tmp2.threadId)
	y = tmp2.value
	fmt.Println(y)
	tracer.ClosePrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:20")
	close(x)
	tracer.CloseCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:20")
	close(x)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:22")
	tmp3 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:22", tmp3.threadId)

	go func() {
		tracer.RegisterThread("fun2")
		tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:25")
		tmp4 := <-x
		tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\manual.go:25", tmp4.threadId)
	}()

}
