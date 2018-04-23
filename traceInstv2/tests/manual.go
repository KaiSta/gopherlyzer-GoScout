package main

import "fmt"

func foo(x chan int, y int) {
	x <- y
}

func main() {
	x := make(chan int)
	go foo(x, 1)
	go foo(x, 2)

	y := <-x
	fmt.Println(y)

	y = <-x
	fmt.Println(y)

	close(x)

	<-x

	go func() {
		<-x
	}()
}
