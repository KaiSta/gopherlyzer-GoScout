package main

func foo(x chan int, y chan int) {
	x <- 1
	y <- 1
}

func bar(y chan int) {
	y <- 2
}

func main() {
	x := make(chan int, 1)
	y := make(chan int)

	go foo(x, y)
	go bar(y)
	go baz(x)

	<-y
	<-y
}
