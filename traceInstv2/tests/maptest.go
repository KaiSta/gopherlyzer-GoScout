package main

import "fmt"

func main() {
	m := make(map[int]int)
	c := make(chan int)

	x := m[0]
	x++
	m[0] = x

	fmt.Println(m[0])

	z := make([]int, 2)
	z[1] = 1
	z[0] = 0
	fmt.Println(z[0])
	fmt.Println(z)
}
