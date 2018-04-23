package main

import (
	"fmt"
	"sync"
	"time"
)

var x = 0

func reader(m *sync.Mutex, c chan int) {
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		m.Lock()
		fmt.Sprint(x)
		m.Unlock()
	}
	c <- 1
}
func writer(a, b, c, d *sync.Mutex, y chan int) {
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		a.Lock()
		b.Lock()
		c.Lock()
		d.Lock()
		x++
		d.Unlock()
		c.Unlock()
		b.Unlock()
		a.Unlock()
	}
	y <- 1
}

func main() {
	a := &sync.Mutex{}
	b := &sync.Mutex{}
	c := &sync.Mutex{}
	d := &sync.Mutex{}

	ch := make(chan int)

	go reader(a, ch)
	go reader(b, ch)
	go reader(c, ch)
	go reader(d, ch)

	go writer(a, b, c, d, ch)
	go writer(a, b, c, d, ch)

	<-ch
	<-ch
	<-ch
	<-ch
	<-ch
	<-ch
}
