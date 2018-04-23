package main

import (
	"fmt"
	"time"
)

func generate(ch chan int) {
	for i := 2; ; i++ {
		ch <- i
	}
}

func filter(in chan int, out chan int, prime int) {
	for {
		tmp := <-in
		if tmp%prime != 0 {
			out <- tmp
		}
	}
}

func main() {
	start := time.Now()
	ch := make(chan int)
	go generate(ch)
	for i := 0; i < 100; i++ {
		prime := <-ch
		//fmt.Println(prime)
		ch1 := make(chan int)
		go filter(ch, ch1, prime)
		ch = ch1
	}
	fmt.Println(time.Since(start))
}
