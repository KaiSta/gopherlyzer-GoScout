package main

func foo(x chan int, y chan int) {
	<-x
	y <- 1
}
func bar(x chan int, y chan int) {
	<-x
	y <- 2
}

func main() {
	x := make(chan int, 2)
	y := make(chan int)
	go foo(x, y)
	go bar(x, y)
	x <- 1
	x <- 2
	<-y
	<-y
}
