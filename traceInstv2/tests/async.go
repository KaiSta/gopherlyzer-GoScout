package main

var z int

func foo(x chan int, y chan int) {
	x <- 1
	z++
	<-x
	y <- 1
}

func main() {
	x := make(chan int, 1)
	y := make(chan int)
	go foo(x, y)
	go foo(x, y)

	<-y
	<-y
}
