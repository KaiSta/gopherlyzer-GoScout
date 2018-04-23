package main

import "fmt"

func foo(v int, w string) {
	fmt.Println(v, w)
}

func main() {
	x := make(chan int)
	y := make(chan string)

	foo(<-x, <-y)
}
