package main

import "../vc"

func foo(x *vc.ChanInt, vc vc.VectorClock) {
	x.Send(42, vc)
}

func main() {
	vc.Start()
	mainVC := vc.NewVC()
	x := vc.NewChanInt("x")

	go foo(x, vc.NewVC())
	go foo(x, vc.NewVC())

	x.Rcv(mainVC)
	x.Rcv(mainVC)

	vc.Stop()
}
