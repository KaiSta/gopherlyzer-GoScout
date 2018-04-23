package main

import (
	"fmt"
	"time"

	"../traceInst/tracer"
)

func generate(ch chan struct {
	threadId uint64
	value    int
}) {
	for i := 2; ; i++ {
		tracer.PrepSend(ch, "103")
		ch <- struct {
			threadId uint64
			value    int
		}{tracer.GetGID(), i}
		tracer.PostSend(ch, "103")
	}
}

func filter(in chan struct {
	threadId uint64
	value    int
}, out chan struct {
	threadId uint64
	value    int
}, prime int) {
	for {
		tracer.PrepRcv(in, "191")
		tmp1 := <-in
		tracer.PostRcv(in, "191", tmp1.threadId)
		tmp := tmp1.value
		if tmp%prime != 0 {
			tracer.PrepSend(out, "223")
			out <- struct {
				threadId uint64
				value    int
			}{tracer.GetGID(), tmp}
			tracer.PostSend(out, "223")
		}
	}
}

func main() {
	tracer.Start()
	start := time.Now()
	ch := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(ch, cap(ch))
	tmp2 := tracer.GetWaitSigID()
	tracer.AddSignal(tmp2)
	go func(ch chan struct {
		threadId uint64
		value    int
	}) {
		tracer.RegisterThread("generate0", tmp2)

		generate(ch)
	}(ch)
	for i := 0; i < 500; i++ {
		tracer.PrepRcv(ch, "367")
		tmp3 := <-ch
		tracer.PostRcv(ch, "367", tmp3.threadId)
		prime := tmp3.value

		ch1 := make(chan struct {
			threadId uint64
			value    int
		})
		tracer.RegisterChan(ch1, cap(ch1))
		tmp4 := tracer.GetWaitSigID()
		tracer.AddSignal(tmp4)
		go func(ch chan struct {
			threadId uint64
			value    int
		},

			ch1 chan struct {
				threadId uint64
				value    int
			},

			prime int) {
			tracer.RegisterThread("filter1", tmp4)

			filter(ch, ch1, prime)
		}(ch, ch1, prime)
		ch = ch1
	}
	fmt.Println(time.Since(start))
	tracer.Stop()
}
