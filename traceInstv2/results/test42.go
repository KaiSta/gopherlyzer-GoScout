package main

import "fmt"

func foo(v int, w string) {
	fmt.Println(v, w)
}

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	y := make(chan struct {
		threadId uint64
		value    string
	})
	tracer.RegisterChan(y, cap(y))
	tracer.RcvPrep(x, "c:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\fun.go:13")
	tmp1 := <-x
	tracer.RcvCommit(x, "c:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\fun.go:13", tmp1.threadId)
	tracer.RcvPrep(y, "c:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\fun.go:13")
	tmp2 := <-y
	tracer.RcvCommit(y, "c:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\fun.go:13", tmp2.threadId)
	foo(tmp1, tmp2)
}
