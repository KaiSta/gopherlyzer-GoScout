package main

func foo(x chan int) {
	x <- 1
}

func main() {
	x := make(chan int)
	go foo(x)
	go foo(x)
	<-x
}
