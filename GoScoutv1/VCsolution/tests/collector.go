package main

import (
	"fmt"
	"time"

	"../vc"
)

func collect(x *vc.ChanInt, v int, vc vc.VectorClock) {
	x.Send(v, vc)
}

func main() {
	mainVC := vc.NewVC()
	vc.Start()
	x := vc.NewChanInt("x")
	start := time.Now()
	for i := 0; i < 1000; i++ {
		go collect(x, i, vc.NewVC())
	}

	for i := 0; i < 1000; i++ {
		x.Rcv(mainVC)
	}
	fmt.Println(time.Since(start))
	vc.Stop()
}
