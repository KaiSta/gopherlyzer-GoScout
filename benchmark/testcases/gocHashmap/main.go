package main

import (
	"fmt"
	"sync"
)

func writer(m *sync.Map, l, h int, c chan int) {
	for i := l; i < h; i++ {
		m.Store(i, i)
	}
	c <- 1
}
func reader(m *sync.Map, l, h int, c chan int) {
	for i := l; i < h; {
		v, ok := m.Load(i)
		if ok {
			i++
		}
		fmt.Sprint(v)
	}
	c <- 1
}

func main() {
	m := sync.Map{}
	c := make(chan int)
	go writer(&m, 0, 100, c)
	go writer(&m, 0, 100, c)
	go writer(&m, 0, 100, c)

	go reader(&m, 0, 100, c)
	go reader(&m, 0, 100, c)

	<-c
	<-c
	<-c
	<-c
	<-c

}
