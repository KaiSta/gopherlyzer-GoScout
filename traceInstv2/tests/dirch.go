package main

func foo(x chan<- int) {

}

func main() {
	x := make(chan int)
	foo(x)
}
