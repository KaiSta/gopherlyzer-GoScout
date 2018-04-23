package main

import (
	"fmt"
	"time"

	"../../traceInst/tracer2"
	"github.com/pkg/profile"
)

func generate(ch chan struct {
	threadId uint64
	value    int
}) {
	for i := 2; ; i++ {
		tracer2.PrepSend(ch, "103")
		ch <- struct {
			threadId uint64
			value    int
		}{tracer2.GetGID(), i}
		tracer2.PostSend(ch, "103")
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
		tracer2.PrepRcv(in, "191")
		tmp1 := <-in
		tracer2.PostRcv(in, "191", tmp1.threadId)
		tmp := tmp1.value
		if tmp%prime != 0 {
			tracer2.PrepSend(out, "223")
			out <- struct {
				threadId uint64
				value    int
			}{tracer2.GetGID(), tmp}
			tracer2.PostSend(out, "223")
		}
	}
}

func main() {
	// f, _ := os.Create("mylog.prof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	defer profile.Start().Stop()
	tracer2.Start()
	start := time.Now()
	ch := make(chan struct {
		threadId uint64
		value    int
	})
	tracer2.RegisterChan(ch, cap(ch))
	tmp2 := tracer2.GetWaitSigID()
	tracer2.AddSignal(tmp2)
	go func(ch chan struct {
		threadId uint64
		value    int
	}) {
		tracer2.RegisterThread("generate0", tmp2)

		generate(ch)
		tracer2.StopThread()
	}(ch)
	for i := 0; i < 1000; i++ {
		tracer2.PrepRcv(ch, "367")
		tmp3 := <-ch
		tracer2.PostRcv(ch, "367", tmp3.threadId)
		prime := tmp3.value

		ch1 := make(chan struct {
			threadId uint64
			value    int
		})
		tracer2.RegisterChan(ch1, cap(ch1))
		tmp4 := tracer2.GetWaitSigID()
		tracer2.AddSignal(tmp4)
		go func(ch chan struct {
			threadId uint64
			value    int
		},

			ch1 chan struct {
				threadId uint64
				value    int
			},

			prime int) {
			tracer2.RegisterThread("filter1", tmp4)

			filter(ch, ch1, prime)
			tracer2.StopThread()

		}(ch, ch1, prime)
		ch = ch1
	}
	fmt.Println(time.Since(start))
	tracer2.Stop()
}
