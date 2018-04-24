package main

import "../tracer"

func main() {
	tracer.Start()
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(c, "tests\\smallEx.go:6", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "tests\\smallEx.go:6", myTIDCache)
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(c, "tests\\smallEx.go:9", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "tests\\smallEx.go:9", myTIDCache)
	}()
	tracer.PreRcv(c, "tests\\smallEx.go:12", myTIDCache)
	tmp3 := <-c
	tracer.PostRcv(c, "tests\\smallEx.go:12", tmp3.threadId, myTIDCache)
	tracer.Stop()
}
