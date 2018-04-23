package main

import "fmt"

// func foo(x int) {
// 	x = x + x - g
// }

var g int

func main() {
	// x := 0
	// y := 1

	// x++

	// y--

	// z := x + y

	// foo(x)

	// ch := make(chan int)

	// ch <- x
	// z = <-ch + y
	var y string
	var z = 0

	g++
	fmt.Println(y, z, g)
}
