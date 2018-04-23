package main

import (
	"fmt"
	"time"
)

func add1(in chan int) chan int {
	out := make(chan int)
	go func() {
		for {
			n := <-in
			// if n == 0 {
			// 	close(out)
			// 	return
			// }
			out <- n + 1
		}
	}()
	return out
}

func main() {
	start := time.Now()
	in := make(chan int)
	c1 := add1(in)

	for i := 0; i < 49; i++ {
		c1 = add1(c1)
	}

	for n := 1; n < 1000; n++ {
		in <- n
		<-c1
	}
	fmt.Println(time.Since(start))
}
