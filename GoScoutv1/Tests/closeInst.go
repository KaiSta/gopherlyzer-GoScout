package main

func foo(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.SendPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:4")
	x <- struct {
		threadId uint64
		value    int
	}{tracer.GetGID(), 1}
	tracer.SendCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:4")
}
func bar(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.RcvPrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:7")
	tmp1 := <-x
	tracer.RcvCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:7", tmp1.threadId)
}
func baz(x chan struct {
	threadId uint64
	value    int
}) {
	tracer.ClosePrep(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:10")
	close(x)
	tracer.CloseCommit(x, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\Tests\\close.go:10")
}
func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
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
	tmp3 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp3)
	go func(x chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("baz1", tmp3)

		baz(x)
	}(x)

}
