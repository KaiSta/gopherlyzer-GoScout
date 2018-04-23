package main

import (
	"fmt"
	"sync"
)

func foo(x *sync.Mutex) {

}

func bar(x, y int) (int, int) {
	x++
	y++
	fmt.Println(x)
	return x, y
}

func baz() int {
	return 42
}

func main() {
	a := sync.Mutex{}

	go foo(&a)
	x := baz()
	v1, v2 := bar(5, x)

	fmt.Println(v1, v2)

	v1, _ = bar(4, 5)

	y := make([]int, 2)
	v3 := y[0]
}
