package main

type SU struct{}

func main() {
	var x = make(chan int)

	<-x

	x <- 1

	var y = 5

	var z = make(chan struct {
		x int
	})
	<-z
}
