package main

import "fmt"

func foo(x int) int {
	return 42 + x
}

func main() {
	x := make(chan int)
	<-x
	x <- 1

	z := 8
	y := 4
	y = y + z + foo(y)
	fmt.Println(y, z)
}

// import "fmt"

// func foo(x, y chan int) {
// 	<-x
// 	y <- 1
// }

// func main() {
// 	x := make(chan int)
// 	y := make(chan int)
// 	go foo(x, y)
// 	x <- 1
// 	<-y
// 	fmt.Println("done")
// }

// // go func() {
// // 	tracer.RegisterThread(..)
// // 	foo(x,y)
// // }
