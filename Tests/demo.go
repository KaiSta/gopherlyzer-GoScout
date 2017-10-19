package main

import "fmt"

func A(x chan int) {
	x <- 1
}

func B(x chan int) {
	x <- 2
}

func main() {
	x := make(chan int)

	go A(x)
	go B(x)

	fmt.Println(<-x)
}
