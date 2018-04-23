package main

import "fmt"
import "../vc"
import "time"

func generate(ch *vc.ChanInt, vclock vc.VectorClock) {
	for i := 2; ; i++ {
		ch.Send(i, vclock)
	}
}

func filter(in *vc.ChanInt, out *vc.ChanInt, prime int, vclock vc.VectorClock) {
	for {
		tmp := in.Rcv(vclock)
		if tmp%prime != 0 {
			out.Send(tmp, vclock)
		}
	}
}

func main() {
	start := time.Now()
	vc.Start()
	mainVC := vc.NewVC()
	ch := vc.NewChanInt("gen")
	go generate(ch, vc.NewVC())
	for i := 0; i < 500; i++ {
		prime := ch.Rcv(mainVC)
		//fmt.Println(prime)
		ch1 := vc.NewChanInt(fmt.Sprintf("ch1-%v", i))
		go filter(ch, ch1, prime, vc.NewVC())
		ch = ch1
	}
	vc.Stop()
	fmt.Println(time.Since(start))
}
