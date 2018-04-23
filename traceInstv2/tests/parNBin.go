package main

func main() {
	x := make(chan int)

	u := (5)

	w := <-x + <-x

	v := (<-x) * (<-x + <-x)
}
