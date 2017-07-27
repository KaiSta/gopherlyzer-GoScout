package main

import "fmt"

func foo(x chan int, y chan string) {
	<-x
	v := <-x
	fmt.Println(v, &x)

	for i := 0; i < 10; i++ {
		if 5 < 8 {
			y <- "hello"
		}

	}

}

func main() {
	x := make(chan int)
	y := make(chan string, 1)
	val := 42
	go foo(x, y)
	x <- val

	if 6 > 8 {
		<-x
	} else if 5 < 9 {
		x <- 5
	}

	select {
	case <-x:
	case y <- "a":
	case vv := <-x:
		fmt.Println(vv)
	}
}
