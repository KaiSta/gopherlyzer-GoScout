package main

import (
	"fmt"
	"time"
)

func reuters(ch chan string) {
	ch <- "REUTERS"

}

func bloomberg(ch chan string) {
	ch <- "BLOOMBERG"

}

// with helper threads
func newsReader(reutersCh chan string, bloombergCh chan string) {
	ch := make(chan string)

	go func() {
		ch <- <-reutersCh
	}()

	go func() {
		ch <- <-bloombergCh
	}()

	x := <-ch
	fmt.Printf("got news from %s \n", x)

}

func main() {
	start := time.Now()
	reutersCh := make(chan string)
	bloombergCh := make(chan string)

	go reuters(reutersCh)
	go bloomberg(bloombergCh)
	go newsReader(reutersCh, bloombergCh)
	// newsReader(reutersCh, bloombergCh) // in most cases deadlock
	newsReader(reutersCh, bloombergCh)
	fmt.Println(time.Since(start))
}
