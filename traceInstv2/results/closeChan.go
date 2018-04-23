package main

import (
	"time"

	"../tracer"
)

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		tracer.PreSend(c, "tests\\closeChan.go:7", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(c, "tests\\closeChan.go:7", tracer.GetGID())
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		tracer.PreRcv(c, "tests\\closeChan.go:10", tracer.GetGID())
		tmp3 := <-c
		tracer.PostRcv(c, "tests\\closeChan.go:10", tmp3.threadId, tracer.GetGID())
	}()
	time.Sleep(1 * time.Second)
	tracer.PreClose(c, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\closeChan.go:12", tracer.GetGID())
	close(c)
	tracer.PostClose(c, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\closeChan.go:12", tracer.GetGID())
	tracer.Stop()
}
