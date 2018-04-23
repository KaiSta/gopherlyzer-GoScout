package main

import "../tracer"

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	ch := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(ch, cap(ch))
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
		tracer.PreSend(ch, "tests\\altComm.go:8", tracer.GetGID())
		ch <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(ch, "tests\\altComm.go:8", tracer.GetGID())
		tracer.PreSend(c, "tests\\altComm.go:9", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(c, "tests\\altComm.go:9", tracer.GetGID())
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		tracer.PreSend(ch, "tests\\altComm.go:12", tracer.GetGID())
		ch <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 2}
		tracer.PostSend(ch, "tests\\altComm.go:12", tracer.GetGID())
		tracer.PreSend(c, "tests\\altComm.go:13", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 2}
		tracer.PostSend(c, "tests\\altComm.go:13", tracer.GetGID())
	}()
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun2")
		tracer.Wait(tmp3, tracer.GetGID())
		tracer.PreRcv(ch, "tests\\altComm.go:16", tracer.GetGID())
		tmp4 := <-ch
		tracer.PostRcv(ch, "tests\\altComm.go:16", tmp4.threadId, tracer.GetGID())
		tracer.PreRcv(ch, "tests\\altComm.go:17", tracer.GetGID())
		tmp5 := <-ch
		tracer.PostRcv(ch, "tests\\altComm.go:17", tmp5.threadId, tracer.GetGID())
		tracer.PreSend(c, "tests\\altComm.go:18", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 3}
		tracer.PostSend(c, "tests\\altComm.go:18", tracer.GetGID())
	}()
	tracer.PreRcv(c, "tests\\altComm.go:21", tracer.GetGID())
	tmp6 := <-c
	tracer.PostRcv(c, "tests\\altComm.go:21", tmp6.threadId, tracer.GetGID())
	tracer.PreRcv(c, "tests\\altComm.go:22", tracer.GetGID())
	tmp7 := <-c
	tracer.PostRcv(c, "tests\\altComm.go:22", tmp7.threadId, tracer.GetGID())
	tracer.PreRcv(c, "tests\\altComm.go:23", tracer.GetGID())
	tmp8 := <-c
	tracer.PostRcv(c, "tests\\altComm.go:23", tmp8.threadId, tracer.GetGID())
	tracer.Stop()
}
