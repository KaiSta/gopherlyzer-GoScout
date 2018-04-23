package main

import "fmt"

func main() {
	x := make(chan int)
	y := make(chan int)

	go func() {
		x <- 1
	}()
	go func() {
		y <- 1
	}()

	select {
	case <-x:
		fmt.Println("rcv x")
	case <-y:
		fmt.Println("rcv y")
	}
}
