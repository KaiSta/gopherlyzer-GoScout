package main

import (
	"fmt"
	"sync"
	"time"

	"../../tracer"
)

var x = 0

func reader(m *sync.Mutex, c chan struct {
	threadId uint64
	value    int
}) {
	myTIDCache := tracer.GetGID()
	for i := 0; i < 100; i++ {
		myTIDCache := tracer.GetGID()
		time.Sleep(1 * time.Second)
		tracer.PreLock(&m, "perfBench\\ex1\\main.go:14", myTIDCache)
		m.Lock()
		tracer.ReadAcc(&x, "perfBench\\ex1\\main.go:15", myTIDCache)
		fmt.Sprint(x)
		tracer.PostLock(&m, "perfBench\\ex1\\main.go:16", myTIDCache)
		m.Unlock()
	}
	tracer.PreSend(c, "perfBench\\ex1\\main.go:18", myTIDCache)
	c <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 1}
	tracer.PostSend(c, "perfBench\\ex1\\main.go:18", myTIDCache)
}
func writer(a, b, c, d *sync.Mutex, y chan struct {
	threadId uint64
	value    int
}) {
	myTIDCache := tracer.GetGID()
	for i := 0; i < 100; i++ {
		time.Sleep(1 * time.Second)
		tracer.PreLock(&a, "perfBench\\ex1\\main.go:23", myTIDCache)
		a.Lock()
		tracer.PreLock(&b, "perfBench\\ex1\\main.go:24", myTIDCache)
		b.Lock()
		tracer.PreLock(&c, "perfBench\\ex1\\main.go:25", myTIDCache)
		c.Lock()
		tracer.PreLock(&d, "perfBench\\ex1\\main.go:26", myTIDCache)
		d.Lock()
		tracer.WriteAcc(&x, "perfBench\\ex1\\main.go:27", myTIDCache)
		x++
		tracer.PostLock(&d, "perfBench\\ex1\\main.go:28", myTIDCache)
		d.Unlock()
		tracer.PostLock(&c, "perfBench\\ex1\\main.go:29", myTIDCache)
		c.Unlock()
		tracer.PostLock(&b, "perfBench\\ex1\\main.go:30", myTIDCache)
		b.Unlock()
		tracer.PostLock(&a, "perfBench\\ex1\\main.go:31", myTIDCache)
		a.Unlock()
	}
	tracer.PreSend(y, "perfBench\\ex1\\main.go:33", myTIDCache)
	y <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 1}
	tracer.PostSend(y, "perfBench\\ex1\\main.go:33", myTIDCache)
}

func main() {
	tracer.Start()
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	a := &sync.Mutex{}
	tracer.WriteAcc(&a, "perfBench\\ex1\\main.go:37", myTIDCache)
	b := &sync.Mutex{}
	tracer.WriteAcc(&b, "perfBench\\ex1\\main.go:38", myTIDCache)
	c := &sync.Mutex{}
	tracer.WriteAcc(&c, "perfBench\\ex1\\main.go:39", myTIDCache)
	d := &sync.Mutex{}
	tracer.WriteAcc(&d, "perfBench\\ex1\\main.go:40", myTIDCache)

	ch := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(ch, cap(ch))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func(m *sync.Mutex, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader0")
		tracer.Wait(tmp1, tracer.GetGID())

		reader(m, c)
	}(a, ch)
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)
	go func(m *sync.Mutex, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader1")
		tracer.Wait(tmp2, tracer.GetGID())

		reader(m, c)
	}(b, ch)
	tmp3 := tracer.GetWaitSigID()
	tracer.Signal(tmp3, myTIDCache)
	go func(m *sync.Mutex, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader2")
		tracer.Wait(tmp3, tracer.GetGID())

		reader(m, c)
	}(c, ch)
	tmp4 := tracer.GetWaitSigID()
	tracer.Signal(tmp4, myTIDCache)
	go func(m *sync.Mutex, c chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("reader3")
		tracer.Wait(tmp4, tracer.GetGID())

		reader(m, c)
	}(d, ch)
	tmp5 := tracer.GetWaitSigID()
	tracer.Signal(tmp5, myTIDCache)

	go func(a, b, c, d *sync.Mutex, y chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("writer4")
		tracer.Wait(tmp5, tracer.GetGID())

		writer(a, b, c, d, y)
	}(a, b, c, d, ch)
	tmp6 := tracer.GetWaitSigID()
	tracer.Signal(tmp6, myTIDCache)
	go func(a, b, c, d *sync.Mutex, y chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("writer5")
		tracer.Wait(tmp6, tracer.GetGID())

		writer(a, b, c, d, y)
	}(a, b, c, d, ch)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:52", myTIDCache)
	tmp7 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:52", tmp7.threadId, myTIDCache)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:53", myTIDCache)
	tmp8 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:53", tmp8.threadId, myTIDCache)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:54", myTIDCache)
	tmp9 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:54", tmp9.threadId, myTIDCache)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:55", myTIDCache)
	tmp10 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:55", tmp10.threadId, myTIDCache)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:56", myTIDCache)
	tmp11 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:56", tmp11.threadId, myTIDCache)
	tracer.PreRcv(ch, "perfBench\\ex1\\main.go:57", myTIDCache)
	tmp12 := <-ch
	tracer.PostRcv(ch, "perfBench\\ex1\\main.go:57", tmp12.threadId, myTIDCache)
	tracer.Stop()
}
