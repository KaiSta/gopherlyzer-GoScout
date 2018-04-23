package main

var y int

func foo(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\racetest.go:6")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\racetest.go:6")
	y++
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\racetest.go:8")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\racetest.go:8", tmp1.threadId)
}

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	}, 1)
	tracer.RegisterChan(x, cap(x))
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(x chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("foo0", tmp2)

		foo(x)
	}(x)
	tracer.ReadAcc(x)
	foo(x)
}
