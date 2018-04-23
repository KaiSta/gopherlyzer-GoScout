package main

import (
	"fmt"
	"time"

	"../../traceInst/tracer3"
	"github.com/pkg/profile"
)

func generate(ch chan struct {
	threadId uint64
	value    int
}) {
	for i := 2; ; i++ {
		tracer3.PrepSend(ch, "103")
		ch <- struct {
			threadId uint64
			value    int
		}{tracer3.GetGID(), i}
		tracer3.PostSend(ch, "103")
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
		tracer3.PrepRcv(in, "191")
		tmp1 := <-in
		tracer3.PostRcv(in, "191", tmp1.threadId)
		tmp := tmp1.value
		if tmp%prime != 0 {
			tracer3.PrepSend(out, "223")
			out <- struct {
				threadId uint64
				value    int
			}{tracer3.GetGID(), tmp}
			tracer3.PostSend(out, "223")
		}
	}
}

func main() {
	// f, _ := os.Create("mylog.prof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	defer profile.Start().Stop()
	tracer3.Start()
	start := time.Now()
	ch := make(chan struct {
		threadId uint64
		value    int
	})
	tracer3.RegisterChan(ch, cap(ch))
	tmp2 := tracer3.GetWaitSigID()
	tracer3.AddSignal(tmp2)
	go func(ch chan struct {
		threadId uint64
		value    int
	}) {
		tracer3.RegisterThread("generate0", tmp2)

		generate(ch)
	}(ch)
	for i := 0; i < 1000; i++ {
		tracer3.PrepRcv(ch, "367")
		tmp3 := <-ch
		tracer3.PostRcv(ch, "367", tmp3.threadId)
		prime := tmp3.value

		ch1 := make(chan struct {
			threadId uint64
			value    int
		})
		tracer3.RegisterChan(ch1, cap(ch1))
		tmp4 := tracer3.GetWaitSigID()
		tracer3.AddSignal(tmp4)
		go func(ch chan struct {
			threadId uint64
			value    int
		},

			ch1 chan struct {
				threadId uint64
				value    int
			},

			prime int) {
			tracer3.RegisterThread("filter1", tmp4)

			filter(ch, ch1, prime)

		}(ch, ch1, prime)
		ch = ch1
	}
	fmt.Println(time.Since(start))
	tracer3.Stop()
}
