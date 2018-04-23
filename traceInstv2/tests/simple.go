package main

func foo(x chan int, y int) {
	x <- y
	x <- 1
	x <- 2
	x <- 3
}

func main() {
	x := make(chan int)
	go foo(x, 42)
	<-x
	z := (<-x) + <-x
}
