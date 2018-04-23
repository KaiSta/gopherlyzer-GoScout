package main

import "../tracer"

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
		value    string
	})
	tracer.RegisterChan(y, cap(y))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		tracer.PreSend(x, "tests\\wSelect.go:8", tracer.GetGID())
		x <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), 1}
		tracer.PostSend(x, "tests\\wSelect.go:8", tracer.GetGID())
	}()
	tracer.PreSelect(tracer.GetGID(), tracer.SelectEv{x, "?", "tests\\wSelect.go:11"}, tracer.SelectEv{y, "?", "tests\\wSelect.go:11"}, tracer.SelectEv{nil, "?", "tests\\wSelect.go:11"})
	select {
	case tmp2 := <-x:
		tracer.PostRcv(x, "tests\\wSelect.go:12", tmp2.threadId, tracer.GetGID())
	case tmp3 := <-y:
		tracer.PostRcv(y, "tests\\wSelect.go:13", tmp3.threadId, tracer.GetGID())
	default:
		tracer.PostRcv(nil, "tests\\wSelect.go:14", 0, tracer.GetGID())
	}
	tracer.Stop()
}
