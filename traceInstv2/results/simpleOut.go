package main

import (
	"sync"
)

func foo(x chan int, y int) {
	tracer.ReadAcc(&y, "tests\\simple.go:8", tracer.GetGID())
	tracer.PreSend(x, "tests\\simple.go:8", tracer.GetGID())
	x <- y
	tracer.PreSend(x, "tests\\simple.go:9", tracer.GetGID())
	x <- 1
	tracer.PreSend(x, "tests\\simple.go:10", tracer.GetGID())
	x <- 2
	tracer.PreSend(x, "tests\\simple.go:11", tracer.GetGID())
	x <- 3
}

func main() {
	tracer.RegisterThread("main")
	x := make(chan int)
	tracer.RegisterChan(x, cap(x))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())
	go func(x chan int, y int) {
		tracer.RegisterThread("foo0")
		tracer.Wait(tmp1, tracer.GetGID())

		foo(x, 42)
	}(x, 42)
	tracer.PreRcv(x, "tests\\simple.go:17", tracer.GetGID())
	<-x
	tracer.PreRcv(x, "tests\\simple.go:18", tracer.GetGID())
	tracer.PreRcv(x, "tests\\simple.go:18", tracer.GetGID())
	z := (<-x) + <-x
	tracer.WriteAcc(&z, "tests\\simple.go:18", tracer.GetGID())

	m := sync.Mutex{}
	tracer.PreLock(m, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\simple.go:22")
	m.Lock()
	tracer.WriteAcc(&z, "tests\\simple.go:23", tracer.GetGID())
	z++
	tracer.PostLock(m, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\simple.go:24")
	m.Unlock()
}
