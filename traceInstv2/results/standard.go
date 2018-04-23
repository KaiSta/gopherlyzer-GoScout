package main

import "../tracer"

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	ch := make(chan struct {
		threadId uint64
		value    int
	}, 1)
	tracer.RegisterChan(ch, cap(ch))
	x := 0
	tracer.WriteAcc(&x, "tests\\standard.go:5", tracer.GetGID())
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
		tracer.WriteAcc(&x, "tests\\standard.go:8", tracer.GetGID())
		x++
		tracer.PreSend(ch, "tests\\standard.go:9", tracer.GetGID())
		ch <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(ch, "tests\\standard.go:9", tracer.GetGID())
		tracer.PreRcv(ch, "tests\\standard.go:10", tracer.GetGID())
		tmp2 := <-ch
		tracer.PostRcv(ch, "tests\\standard.go:10", tmp2.threadId, tracer.GetGID())
		tracer.PreSend(c, "tests\\standard.go:11", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(c, "tests\\standard.go:11", tracer.GetGID())
	}()
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp3, tracer.GetGID())
		tracer.PreSend(ch, "tests\\standard.go:14", tracer.GetGID())
		ch <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(ch, "tests\\standard.go:14", tracer.GetGID())
		tracer.PreRcv(ch, "tests\\standard.go:15", tracer.GetGID())
		tmp4 := <-ch
		tracer.PostRcv(ch, "tests\\standard.go:15", tmp4.threadId, tracer.GetGID())
		tracer.WriteAcc(&x, "tests\\standard.go:16", tracer.GetGID())
		x++
		tracer.PreSend(c, "tests\\standard.go:17", tracer.GetGID())
		c <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(c, "tests\\standard.go:17", tracer.GetGID())
	}()
	tracer.PreRcv(c, "tests\\standard.go:20", tracer.GetGID())
	tmp5 := <-c
	tracer.PostRcv(c, "tests\\standard.go:20", tmp5.threadId, tracer.GetGID())
	tracer.PreRcv(c, "tests\\standard.go:21", tracer.GetGID())
	tmp6 := <-c
	tracer.PostRcv(c, "tests\\standard.go:21", tmp6.threadId, tracer.GetGID())
	tracer.Stop()
}
