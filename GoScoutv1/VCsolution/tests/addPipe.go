package main

import (
	"fmt"
	"time"

	"../vc"
)

func add1(in *vc.ChanInt, vclock vc.VectorClock) *vc.ChanInt {
	out := vc.NewChanInt("out")
	go func() {
		for {
			n := in.Rcv(vclock)
			//	n := <-in
			// if n == 0 {
			// 	close(out)
			// 	return
			// }
			out.Send(n+1, vclock)
			//out <- n + 1
		}
	}()
	return out
}

func main() {
	vc.Start()
	start := time.Now()
	in := vc.NewChanInt("in")
	mainvc := vc.NewVC()

	c1 := add1(in, vc.NewVC())

	for i := 0; i < 49; i++ {
		c1 = add1(c1, vc.NewVC())
	}

	// c2 := add1(c1, vc.NewVC())
	// c3 := add1(c2, vc.NewVC())
	// c4 := add1(c3, vc.NewVC())
	// c5 := add1(c4, vc.NewVC())
	// c6 := add1(c5, vc.NewVC())
	// c7 := add1(c6, vc.NewVC())
	// c8 := add1(c7, vc.NewVC())
	// c9 := add1(c8, vc.NewVC())
	// c10 := add1(c9, vc.NewVC())

	for n := 1; n < 1000; n++ {
		in.Send(n, mainvc)
		//	in <- n
		c1.Rcv(mainvc)
		//<-c10
	}
	fmt.Println(time.Since(start))
	vc.Stop()
}
