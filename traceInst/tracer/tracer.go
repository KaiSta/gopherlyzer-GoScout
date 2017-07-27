package tracer

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"../cmap"
)

var chans *hashmap.CMapv2
var traces *hashmap.CMapv2
var threads *hashmap.CMapv2
var chanID uint64
var threadID uint64
var watchDog chan struct{}
var done chan struct{}

func init() {
	chans = hashmap.NewCMapv2()
	traces = hashmap.NewCMapv2()
	threads = hashmap.NewCMapv2()
	threads.Store(1, "main")
	watchDog = make(chan struct{})
	done = make(chan struct{})
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func informWatchDog() {
	go func() {
		watchDog <- struct{}{}
	}()
}

func Start() {
	go func() { //watchdog
		for {
			select {
			case <-watchDog:
			case <-done:
				//fmt.Println("DONE")
				writeBack()
				done <- struct{}{}
			case <-time.After(1 * time.Second):
				//fmt.Println("Timeout")
				writeBack()

				<-watchDog
			}
		}
	}()
}

func Stop() {
	done <- struct{}{}
	<-done
}

func writeBack() {
	f, err := os.Create("./trace.log")
	if err != nil {
		panic(err)
	}
	iter := traces.Iterator()
	for iter.HasNext() {
		iter2 := iter.Get()
		for iter2.HasNext() {
			f.WriteString(iter2.Get() + "\n")

			iter2.Next()
		}
		iter.Next()
	}
}

func RegisterChan(x interface{}, c int) {
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	id := atomic.AddUint64(&chanID, 1)
	chans.Store(addr, fmt.Sprintf("%v,%v", id, c))
}

func RegisterThread(s string) {
	thread := GetGID()
	threads.Store(thread, fmt.Sprintf("%v%v", s, atomic.LoadUint64(&threadID)))
	atomic.AddUint64(&threadID, 1)
}

func SendPrep(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	informWatchDog()
}

func SendCommit(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}
	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "C", "-"))

	informWatchDog()
}

func RcvPrep(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "P", "-"))

	informWatchDog()
}

func RcvCommit(x interface{}, s string, p uint64) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	pID := "-"
	vec3 := threads.Get(p)
	if vec3 != nil {
		iter2 := vec3.Iterator()
		for iter2.HasNext() {
			pID = iter2.Get()
			iter2.Next()
		}
	}
	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "C", pID))

	informWatchDog()
}

type SelectEv struct {
	X  interface{}
	Op string
	S  string
}

func SelectPrep(evs ...SelectEv) {
	thread := GetGID()
	chanevs := ""
	for i, x := range evs {
		// v := reflect.ValueOf(x.X)
		// addr := uint64(v.Pointer())
		// vec := chans.Get(addr)
		// var chanID string
		// iter := vec.Iterator()
		// for iter.HasNext() {
		// 	chanID = iter.Get()
		// 	iter.Next()
		// }

		var chanID string

		if x.X != nil {
			v := reflect.ValueOf(x.X)
			addr := uint64(v.Pointer())
			vec := chans.Get(addr)

			iter := vec.Iterator()
			for iter.HasNext() {
				chanID = iter.Get()
				iter.Next()
			}
		} else {
			chanID = "0,0"
		}
		chanevs += fmt.Sprintf("(%v,%v,%v)", chanID, x.Op, x.S)

		if i < len(evs)-1 {
			chanevs += ","
		}
	}
	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	//threadid,chanops,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[%v],%v,%v", threadID, chanevs, "P", 0))

	informWatchDog()
}

func ClosePrep(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "#", s, "P", "-"))

	informWatchDog()
}
func CloseCommit(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)

		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	threadID := "-"
	vec2 := threads.Get(thread)
	if vec2 != nil {
		iter2 := vec2.Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}

	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "#", s, "C", "-"))

	informWatchDog()
}
