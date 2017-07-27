package main

func main() {
	x := make(chan int)
	y := make(chan string)

	select {
	case <-x:
	case z := <-y:
	default:
	}
}
