package main

import (
	"fmt"
	"time"
)

func collect(x chan int, v int) {
	x <- v
}

func main() {
	x := make(chan int)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		go collect(x, i)
	}

	for i := 0; i < 1000; i++ {
		<-x
	}
	fmt.Println(time.Since(start))
}
