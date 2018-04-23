package main

func main() {
	x := 0 //L1

	go func() {
		x++ //L2
	}()
	x++ //L3
}
