package main

func sel(x, y chan bool) {
	z := make(chan bool)
	go func() {
		tmp := <-x
		z <- tmp
	}()
	go func() {
		tmp := <-y
		z <- tmp
	}()
	<-z
}

func main() {
	x := make(chan bool)
	y := make(chan bool)
	go func() {
		x <- true
	}()
	go func() {
		y <- false
	}()
	sel(x, y)
	sel(x, y)
}
