package main

func main() {
	x := make(chan int)
	y := make(chan string)

	go func() {
		x <- 1
	}()

	select {
	case <-x:
	case <-y:
	default:
	}
}
