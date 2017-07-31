package main

import "fmt"

func main() {
	x := make(chan int)

	for v := range x {
		fmt.Println(v)
	}

	y := make(map[int]int)
	for k, v := range y {
		fmt.Println(k, v)
	}

}
