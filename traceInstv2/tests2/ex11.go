package main

func main() {
	ch := make(chan int)
	c := make(chan int)
	x := 0

	go func() {
		x = 5   //L1
		ch <- 1 //L2
		c <- 1
	}()
	go func() {
		<-ch //L3
		x++  //L4
		c <- 1
	}()
	<-c
	<-c
}
