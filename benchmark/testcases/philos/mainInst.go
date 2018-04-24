package main

import (
	"../../tracer"
)

func main() {
	tracer.Start()
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	n := 2
	forks := make(chan struct {
		threadId uint64
		value    int
	}, n)
	tracer.RegisterChan(forks, cap(forks))

	for i := 0; i < n; i++ {
		myTIDCache := tracer.GetGID()
		tracer.PreSend(forks, "main.go:8", myTIDCache)
		forks <- struct {
			threadId uint64
			value    int
		}{myTIDCache, i}
		tracer.PostSend(forks, "main.go:8", myTIDCache)
	}

	for i := 0; i < (n - 1); i++ {
		myTIDCache := tracer.GetGID()
		tmp1 := tracer.GetWaitSigID()
		tracer.Signal(tmp1, myTIDCache)
		go func() {
			tracer.RegisterThread("fun0")
			tracer.Wait(tmp1, tracer.GetGID())
			myTIDCache := tracer.GetGID()
			tracer.PreRcv(forks, "main.go:13", myTIDCache)
			tmp2 := <-forks
			tracer.PostRcv(forks, "main.go:13", tmp2.threadId, myTIDCache)
			tracer.PreRcv(forks, "main.go:14", myTIDCache)
			tmp3 := <-forks
			tracer.PostRcv(forks, "main.go:14", tmp3.threadId, myTIDCache)
			tracer.PreSend(forks, "main.go:16", myTIDCache)
			forks <- struct {
				threadId uint64
				value    int
			}{myTIDCache, 1}
			tracer.PostSend(forks, "main.go:16", myTIDCache)
			tracer.PreSend(forks, "main.go:17", myTIDCache)
			forks <- struct {
				threadId uint64
				value    int
			}{myTIDCache, 2}
			tracer.PostSend(forks, "main.go:17", myTIDCache)
		}()

	}
	tracer.PreRcv(forks, "main.go:20", myTIDCache)
	tmp4 := <-forks
	tracer.PostRcv(forks, "main.go:20", tmp4.threadId, myTIDCache)
	tracer.PreRcv(forks, "main.go:21", myTIDCache)
	tmp5 := <-forks
	tracer.PostRcv(forks, "main.go:21", tmp5.threadId, myTIDCache)
	tracer.PreSend(forks, "main.go:23", myTIDCache)
	forks <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 1}
	tracer.PostSend(forks, "main.go:23", myTIDCache)
	tracer.PreSend(forks, "main.go:24", myTIDCache)
	forks <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 2}
	tracer.PostSend(forks, "main.go:24", myTIDCache)
	tracer.Stop()
}
