package main

func foo(x chan struct {
	threadId uint64
	value    int
}, y chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:4")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:4")
	tracer.SendPrep(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:5")
	y <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:5")
}

func bar(y chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:9")
	y <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 2}
	tracer.SendCommit(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:9")
}

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	}, 1)
	tracer.RegisterChan(x, cap(x))
	y := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(y, cap(y))
	tmp1 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp1)

	go func(x chan struct {
		threadId uint64
		value    int
	},

		y chan struct {
			threadId uint64
			value    int
		}) {
		tracer.RegisterThread("foo0", tmp1)

		foo(x, y)
	}(x, y)
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(y chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("bar1", tmp2)

		bar(y)
	}(y)
	tmp3 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp3)
	go func() {
		tracer.RegisterThread("baz2", tmp3)
		baz(x)
	}(x)
	tracer.RcvPrep(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:20")
	tmp4 := <-y
	tracer.RcvCommit(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:20", tmp4.threadId)
	tracer.RcvPrep(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:21")
	tmp5 := <-y
	tracer.RcvCommit(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\async.go:21", tmp5.threadId)
}
