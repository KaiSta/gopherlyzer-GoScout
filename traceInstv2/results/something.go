package main

import "fmt"

func foo(x int) {
	fmt.Println(x)
}

func main() {
	x := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(x, cap(x))

	foo(<-x)
}
