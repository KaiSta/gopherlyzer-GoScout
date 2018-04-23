package main

func main() {
	c := make(chan int)
	x := 0
	y := 0
	z := 0
	if x > 5 && y < 3 {
		y = 4
	} else if y > 2 {
		z = <-c
	} else if ^z > 0 {
		x = 1
	}
}
