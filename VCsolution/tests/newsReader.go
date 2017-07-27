package main

import "fmt"
import "time"
import "../vc"

func reuters(ch *vc.ChanString, vclock vc.VectorClock) {
	ch.Send("REUTERS", vclock)

}

func bloomberg(ch *vc.ChanString, vclock vc.VectorClock) {
	ch.Send("BLOOMBERG", vclock)

}

// with helper threads
func newsReader(reutersCh *vc.ChanString, bloombergCh *vc.ChanString, vclock vc.VectorClock) {
	ch := vc.NewChanString("tmp")

	go func() {
		tmp := vc.NewVC()
		ch.Send(reutersCh.Rcv(tmp), tmp)
	}()

	go func() {
		tmp := vc.NewVC()
		ch.Send(bloombergCh.Rcv(tmp), tmp)
	}()

	x := ch.Rcv(vclock)
	fmt.Printf("got news from %s \n", x)

}

func main() {
	vc.Start()
	start := time.Now()
	mainVC := vc.NewVC()
	reutersCh := vc.NewChanString("reuters")
	bloombergCh := vc.NewChanString("bloomberg")

	go reuters(reutersCh, vc.NewVC())
	go bloomberg(bloombergCh, vc.NewVC())
	go newsReader(reutersCh, bloombergCh, vc.NewVC())
	// newsReader(reutersCh, bloombergCh) // in most cases deadlock
	newsReader(reutersCh, bloombergCh, mainVC)
	fmt.Println(time.Since(start))
	vc.Stop()
}
