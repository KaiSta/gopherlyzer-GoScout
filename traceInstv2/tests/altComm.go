package main

func main() {
	ch := make(chan int)
	c := make(chan int)

	go func() {
		ch <- 1
		c <- 1
	}()
	go func() {
		ch <- 2
		c <- 2
	}()
	go func() {
		<-ch
		<-ch
		c <- 3
	}()

	<-c
	<-c
	<-c

}
