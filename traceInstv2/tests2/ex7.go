package main

func main() {
	x := 0 //L1

	go func() {
		x++ //L2
		x++ //L5
	}()
	x++ //L3
	x++ //L4
}
