package main

import "fmt"
import "../traceInst/tracer"

func A(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:6")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:6")
}

func B(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:10")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 2}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:10")
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
		tracer.RegisterThread("A0", tmp1)

		A(x)
	}(x)
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(x chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("B1", tmp2)

		B(x)
	}(x)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:19")
	tmp3 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\demo.go:19", tmp3.threadId)
	fmt.Println(tmp3.value)
	tracer.Stop()
}
