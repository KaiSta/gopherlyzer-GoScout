package main

import (
	"fmt"
	"time"

	"../../traceInst/tracer4"
	"github.com/pkg/profile"
)

func generate(ch chan struct {
	threadId uint64
	value    int
}) {
	for i := 2; ; i++ {
		tracer4.PrepSend(fmt.Sprintf("%p%v", ch, cap(ch)), "103")
		ch <- struct {
			threadId uint64
			value    int
		}{tracer4.GetGID(), i}
		tracer4.PostSend(fmt.Sprintf("%p%v", ch, cap(ch)), "103")
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
		tracer4.PrepRcv(fmt.Sprintf("%p%v", in, cap(in)), "191")
		tmp1 := <-in
		tracer4.PostRcv(fmt.Sprintf("%p%v", in, cap(in)), "191", tmp1.threadId)
		tmp := tmp1.value
		if tmp%prime != 0 {
			tracer4.PrepSend(fmt.Sprintf("%p%v", out, cap(out)), "223")
			out <- struct {
				threadId uint64
				value    int
			}{tracer4.GetGID(), tmp}
			tracer4.PostSend(fmt.Sprintf("%p%v", out, cap(out)), "223")
		}
	}
}

func main() {
	// f, _ := os.Create("mylog.prof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()
	defer profile.Start().Stop()
	tracer4.Start()
	start := time.Now()
	ch := make(chan struct {
		threadId uint64
		value    int
	})
	//	tracer4.RegisterChan(ch, cap(ch))
	tmp2 := tracer4.GetWaitSigID()
	tracer4.AddSignal(tmp2)
	go func(ch chan struct {
		threadId uint64
		value    int
	}) {
		tracer4.RegisterThread("generate0", tmp2)

		generate(ch)
	}(ch)
	for i := 0; i < 1000; i++ {
		tracer4.PrepRcv(fmt.Sprintf("%p%v", ch, cap(ch)), "367")
		tmp3 := <-ch
		tracer4.PostRcv(fmt.Sprintf("%p%v", ch, cap(ch)), "367", tmp3.threadId)
		prime := tmp3.value

		ch1 := make(chan struct {
			threadId uint64
			value    int
		})
		//	tracer4.RegisterChan(ch1, cap(ch1))
		tmp4 := tracer4.GetWaitSigID()
		tracer4.AddSignal(tmp4)
		go func(ch chan struct {
			threadId uint64
			value    int
		},

			ch1 chan struct {
				threadId uint64
				value    int
			},

			prime int) {
			tracer4.RegisterThread("filter1", tmp4)

			filter(ch, ch1, prime)

		}(ch, ch1, prime)
		ch = ch1
	}
	fmt.Println(time.Since(start))
	//tracer4.Stop()
}
