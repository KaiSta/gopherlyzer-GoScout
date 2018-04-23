package main

import (
	"sync"
)

var z int

func foo(m *sync.Mutex, y chan int) {
	tracer.PreLock(&m, "tests\\mutexAt.go:10", tracer.GetGID())
	m.Lock()
	tracer.WriteAcc(&z, "tests\\mutexAt.go:11", tracer.GetGID())
	z++
	tracer.PostLock(&m, "tests\\mutexAt.go:12", tracer.GetGID())
	m.Unlock()
	tracer.PreSend(y, "tests\\mutexAt.go:14", tracer.GetGID())
	y <- 1
}

func main() {
	tracer.RegisterThread("main")
	m := sync.Mutex{}

	y := make(chan int)
	tracer.RegisterChan(y, cap(y))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())

	go func(m *sync.Mutex,

		y chan int) {
		tracer.RegisterThread("foo0")
		tracer.Wait(tmp1, tracer.GetGID())

		foo(m, y)
	}(&m, y)
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, tracer.GetGID())
	go func(m *sync.Mutex,

		y chan int) {
		tracer.RegisterThread("foo1")
		tracer.Wait(tmp2, tracer.GetGID())

		foo(m, y)
	}(&m, y)
	tracer.PreRcv(y, "tests\\mutexAt.go:25", tracer.GetGID())
	<-y
	tracer.PreRcv(y, "tests\\mutexAt.go:26", tracer.GetGID())
	<-y

}
