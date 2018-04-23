package main

func main() {
	ch := make(chan int, 1)
	x := 0
	c := make(chan int)
	go func() {
		x++
		ch <- 1
		<-ch
		c <- 1
	}()
	go func() {
		ch <- 1
		<-ch
		x++
		c <- 1
	}()

	<-c
	<-c
}
