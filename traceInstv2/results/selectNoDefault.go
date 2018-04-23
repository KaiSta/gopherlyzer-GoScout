package main

import (
	"fmt"

	"../tracer"
)

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	y := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(y, cap(y))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		tracer.PreSend(x, "tests\\selectNoDefault.go:10", tracer.GetGID())
		x <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(x, "tests\\selectNoDefault.go:10", tracer.GetGID())
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		tracer.PreSend(y, "tests\\selectNoDefault.go:13", tracer.GetGID())
		y <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(y, "tests\\selectNoDefault.go:13", tracer.GetGID())
	}()
	tracer.PreSelect(tracer.GetGID(), tracer.SelectEv{x, "?", "tests\\selectNoDefault.go:16"}, tracer.SelectEv{y, "?", "tests\\selectNoDefault.go:16"})
	select {
	case tmp3 := <-x:
		tracer.PostRcv(x, "tests\\selectNoDefault.go:17", tmp3.threadId, tracer.GetGID())
		fmt.Println("rcv x")
	case tmp4 := <-y:
		tracer.PostRcv(y, "tests\\selectNoDefault.go:19", tmp4.threadId, tracer.GetGID())
		fmt.Println("rcv y")
	}
	tracer.Stop()
}
