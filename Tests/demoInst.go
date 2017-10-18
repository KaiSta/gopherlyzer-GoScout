package main

import "fmt"

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:19")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:19", tmp1.threadId)
	fmt.Println(tmp1.value)
}
