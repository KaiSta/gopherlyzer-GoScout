package main

// func foo(y int) int {
// 	y++
// 	return 42
// }

// type s struct {
// 	x []int
// 	z int
// 	y chan int
// }

// func (t *s) foo(y int) {
// 	return y
// }

type str struct {
	c chan []int
	y int
	x chan int
}

// func foo(x int) int {

// }

func main() {
	s := str{}
	s.c = make(chan []int)
	s.x = make(chan int)

	s.x <- 42

	<-s.x

	select {
	case <-s.x:
	}
}
