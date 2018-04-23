package main

func foo(x chan int) {
	x <- 1
}
func bar(x chan int) {
	<-x
}
func baz2(x chan int) {
	x <- 2
}
func baz(x chan int) {
	close(x)
}
func main() {
	x := make(chan int)
	go foo(x)
	go baz(x)
	bar(x)
}
