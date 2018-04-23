package main

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\partest.go:6")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\partest.go:6", tmp1.threadId)
	v := tmp1.value
}
