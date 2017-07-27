package main

import "fmt"

func foo(x chan int) {
	x <- 1
}

func main() {
	x := make(chan int)
	go foo(x)

	select {
	case <-x:
		fmt.Println("received from x")
	default:
		fmt.Println("default")
	}
}
