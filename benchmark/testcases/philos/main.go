package main

func main() {
	n := 2
	forks := make(chan int, n)

	for i := 0; i < n; i++ {
		forks <- i
	}

	for i := 0; i < (n - 1); i++ {
		go func() {
			<-forks
			<-forks
			//eat
			forks <- 1
			forks <- 2
		}()
	}
	<-forks
	<-forks
	//eat
	forks <- 1
	forks <- 2

}
