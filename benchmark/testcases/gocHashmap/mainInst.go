package main

import (
	"fmt"

	//"../../SpeedyGo/traceInstv2/tracer"
	"../../tracer"
	"./mapInst"
)

func writer(m *maptest.Map, l, h int, c chan struct {
	threadId uint64
	value    int
}) {
	myTIDCache := tracer.GetGID()
	for i := l; i < h; i++ {
		myTIDCache := tracer.GetGID()
		tracer.ReadAcc(&i, "..\\..\\testcases\\gocHashmap\\main.go:12", myTIDCache)
		tracer.ReadAcc(&i, "..\\..\\testcases\\gocHashmap\\main.go:12", myTIDCache)
		m.Store(i, i)
	}
	tracer.PreSend(c, "..\\..\\testcases\\gocHashmap\\main.go:14", myTIDCache)
	c <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 1}
	tracer.PostSend(c, "..\\..\\testcases\\gocHashmap\\main.go:14", myTIDCache)
}
func reader(m *maptest.Map, l, h int, c chan struct {
	threadId uint64
	value    int
}) {
	myTIDCache := tracer.GetGID()
	for i := l; i < h; {
		myTIDCache := tracer.GetGID()
		tracer.ReadAcc(&i, "..\\..\\testcases\\gocHashmap\\main.go:18", myTIDCache)
		v, ok := m.Load(i)
		tracer.WriteAcc(&v, "..\\..\\testcases\\gocHashmap\\main.go:18", myTIDCache)
		tracer.WriteAcc(&ok, "..\\..\\testcases\\gocHashmap\\main.go:18", myTIDCache)
		if ok {
			myTIDCache := tracer.GetGID()
			tracer.WriteAcc(&i, "..\\..\\testcases\\gocHashmap\\main.go:20", myTIDCache)
			i++
		}
		tracer.ReadAcc(&v, "..\\..\\testcases\\gocHashmap\\main.go:22", myTIDCache)
		fmt.Sprint(v)
	}
	tracer.PreSend(c, "..\\..\\testcases\\gocHashmap\\main.go:24", myTIDCache)
	c <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 1}
	tracer.PostSend(c, "..\\..\\testcases\\gocHashmap\\main.go:24", myTIDCache)
}

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	tracer.Start()
	m := maptest.Map{}
	tracer.WriteAcc(&m, "..\\..\\testcases\\gocHashmap\\main.go:29", myTIDCache)
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func(m *maptest.Map, l, h int, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("writer0")
		tracer.Wait(tmp1, tracer.GetGID())

		writer(m, l, h, c)
	}(&m, 0, 100, c)
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)
	go func(m *maptest.Map, l, h int, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("writer1")
		tracer.Wait(tmp2, tracer.GetGID())

		writer(m, l, h, c)
	}(&m, 0, 100, c)
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, myTIDCache)
	go func(m *maptest.Map, l, h int, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("writer2")
		tracer.Wait(tmp3, tracer.GetGID())

		writer(m, l, h, c)
	}(&m, 0, 100, c)
	tmp4 := tracer.GetWaitSigID()
	tracer.Signal(tmp4, myTIDCache)

	go func(m *maptest.Map, l, h int, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader3")
		tracer.Wait(tmp4, tracer.GetGID())

		reader(m, l, h, c)
	}(&m, 0, 100, c)
	tmp5 := tracer.GetWaitSigID()
	tracer.Signal(tmp5, myTIDCache)
	go func(m *maptest.Map, l, h int, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader4")
		tracer.Wait(tmp5, tracer.GetGID())

		reader(m, l, h, c)
	}(&m, 0, 100, c)
	tracer.PreRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:38", myTIDCache)
	tmp6 := <-c
	tracer.PostRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:38", tmp6.threadId, myTIDCache)
	tracer.PreRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:39", myTIDCache)
	tmp7 := <-c
	tracer.PostRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:39", tmp7.threadId, myTIDCache)
	tracer.PreRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:40", myTIDCache)
	tmp8 := <-c
	tracer.PostRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:40", tmp8.threadId, myTIDCache)
	tracer.PreRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:41", myTIDCache)
	tmp9 := <-c
	tracer.PostRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:41", tmp9.threadId, myTIDCache)
	tracer.PreRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:42", myTIDCache)
	tmp10 := <-c
	tracer.PostRcv(c, "..\\..\\testcases\\gocHashmap\\main.go:42", tmp10.threadId, myTIDCache)

	tracer.Stop()
}
