package main

func main() {
	myTIDCache := tracer.GetGID()
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
	x := 0
	tracer.WriteAcc(&x, "ex11.go:6", myTIDCache)
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		x = 5
		tracer.WriteAcc(&x, "ex11.go:9", myTIDCache)
		tracer.PreSend(ch, "ex11.go:10", myTIDCache)
		ch <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(ch, "ex11.go:10", myTIDCache)
		tracer.PreSend(c, "ex11.go:11", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex11.go:11", myTIDCache)
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreRcv(ch, "ex11.go:14", myTIDCache)
		tmp3 := <-ch
		tracer.PostRcv(ch, "ex11.go:14", tmp3.threadId, myTIDCache)
		tracer.WriteAcc(&x, "ex11.go:15", myTIDCache)
		x++
		tracer.PreSend(c, "ex11.go:16", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "ex11.go:16", myTIDCache)
	}()
	tracer.PreRcv(c, "ex11.go:18", myTIDCache)
	tmp4 := <-c
	tracer.PostRcv(c, "ex11.go:18", tmp4.threadId, myTIDCache)
	tracer.PreRcv(c, "ex11.go:19", myTIDCache)
	tmp5 := <-c
	tracer.PostRcv(c, "ex11.go:19", tmp5.threadId, myTIDCache)
}
