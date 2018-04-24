package main

import (
	"sync"
	"time"

	"../../tracer"
)

func main() {
	tracer.Start()
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	a := sync.Mutex{}
	b := sync.Mutex{}
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&a, "main.go:13", myTIDCache)
		a.Lock()
		time.Sleep(1 * time.Second)
		tracer.PreLock(&b, "main.go:15", myTIDCache)
		b.Lock()
		tracer.PostLock(&b, "main.go:16", myTIDCache)
		b.Unlock()
		tracer.PostLock(&a, "main.go:17", myTIDCache)
		a.Unlock()
	}()
	tracer.PreLock(&b, "main.go:20", myTIDCache)
	b.Lock()
	time.Sleep(1 * time.Second)
	tracer.PreLock(&a, "main.go:22", myTIDCache)
	a.Lock()
	tracer.PostLock(&a, "main.go:23", myTIDCache)
	a.Unlock()
	tracer.PostLock(&b, "main.go:24", myTIDCache)
	b.Unlock()
	tracer.Stop()
}
