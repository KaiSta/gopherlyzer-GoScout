package main

var y int

func foo(x chan int) {
	x <- 1
	y++
	<-x
}

func main() {
	x := make(chan int, 1)
	go foo(x)
	foo(x)
}
