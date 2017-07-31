package main

func main() {
	var x = make(chan int)

	<-x

	x <- 1

	var y = 5
}
