package main

type str struct {
	c  struct {
		threadId uint64
		value    []int
	}
	y int
	x  struct {
		threadId uint64
		value    int
	}
}

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	s := str{}
	s.c = make(chan struct {
		threadId uint64
		value    []int
	})
	tracer.RegisterChan(s.c, cap(s.c))
	s.x = make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(s.x, cap(s.x))
	tracer.PreSend(s.x, "tests\\mixed.go:33", myTIDCache)
	s.x <- struct {
		threadId uint64
		value    int
	}{myTIDCache, 42}
	tracer.PostSend(s.x, "tests\\mixed.go:33", myTIDCache)
	tracer.PreRcv(s.x, "tests\\mixed.go:35", myTIDCache)
	tmp1 := <-s.x
	tracer.PostRcv(s.x, "tests\\mixed.go:35", tmp1.threadId, myTIDCache)
	tracer.PreSelect(tracer.GetGID(), tracer.SelectEv{s.x, "?", "tests\\mixed.go:37"})
	select {
	case tmp2 := <-s.x:
		myTIDCache := tracer.GetGID()
		tracer.PostRcv(s.x, "tests\\mixed.go:38", tmp2.threadId, myTIDCache)
	}
}
