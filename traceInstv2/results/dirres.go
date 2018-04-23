package main

func foo(x chan<- struct {
	threadId uint64
	value    int
}) {

}

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))
	foo(x)
}
