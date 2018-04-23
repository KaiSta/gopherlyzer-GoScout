package main

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))

	u := (5)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:8")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:8", tmp1.threadId)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:8")
	tmp2 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:8", tmp2.threadId)
	w := tmp1.value + tmp2.value
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10")
	tmp3 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10", tmp3.threadId)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10")
	tmp4 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10", tmp4.threadId)
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10")
	tmp5 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\parNBin.go:10", tmp5.threadId)
	v := (tmp3.value) * (tmp4.value + tmp5.value)
}
