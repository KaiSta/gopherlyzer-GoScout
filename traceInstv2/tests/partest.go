package main

func main() {
	x := make(chan int)

	v := (<-x)
}
