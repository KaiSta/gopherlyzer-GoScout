package main

func main() {
	x := make(chan int)

	v := <-x + <-x
	z := <-x + 4
	v = 5 + <-x
	k := <-x + <-x - <-x
}
