package tracer2

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var waitSigID uint64
var threadTraces sync.Map
var chanID uint64
var chans sync.Map
var threads sync.Map
var threadFiles sync.Map
var threadID uint64

var threadsExisting uint64

func init() {
	threadTraces = sync.Map{}
	chans = sync.Map{}
	threads = sync.Map{}
	threadFiles = sync.Map{}

	atomic.AddUint64(&threadID, 1)

	threads.Store(uint64(1), threadID)
	// tmp := fmt.Sprintf("%v,[(%v,0,W,-)],C,-", threadID, id)
	// threadTraces.Store(threadID, []string{tmp})

	f, err := os.Create(fmt.Sprintf("%v.log", threadID))
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	threadFiles.Store(threadID, w)

	atomic.AddUint64(&threadID, 1)
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func GetWaitSigID() uint64 {
	return atomic.AddUint64(&waitSigID, 1)
}

func PrepSend(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name, ok := chans.Load(addr)

		if !ok {
			panic("chan not found")
		}

		chanID = name.(string)
	} else {
		chanID = "0,0"
	}

	threadID, ok := threads.Load(thread)
	if !ok {
		panic("thread not found")
	}

	// tmp, ok := threadTraces.Load(threadID)
	// trace := tmp.([]string)
	// trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	// f, ok := threadFiles.Load(threadID)
	// if ok {
	// 	writer := f.(*bufio.Writer)
	// 	writer.WriteString(fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "!", s, "P", "-"))
	// }

	tmp, ok := threadTraces.Load(threadID)
	if ok {
		trace := tmp.([]string)
		trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "!", s, "P", "-"))
		threadTraces.Store(threadID, trace)
	}
}

func PostSend(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name, ok := chans.Load(addr)

		if !ok {
			panic("chan not found")
		}

		chanID = name.(string)
	} else {
		chanID = "0,0"
	}

	threadID, ok := threads.Load(thread)
	if !ok {
		panic("thread not found")
	}

	// tmp, ok := threadTraces.Load(threadID)
	// trace := tmp.([]string)
	// trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	// f, ok := threadFiles.Load(threadID)
	// if ok {
	// 	writer := f.(*bufio.Writer)
	// 	writer.WriteString(fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "!", s, "C", "-"))
	// }

	tmp, ok := threadTraces.Load(threadID)
	if ok {
		trace := tmp.([]string)
		trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "!", s, "C", "-"))
		threadTraces.Store(threadID, trace)
	}
}

func PrepRcv(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name, ok := chans.Load(addr)

		if !ok {
			panic("chan not found")
		}

		chanID = name.(string)
	} else {
		chanID = "0,0"
	}

	threadID, ok := threads.Load(thread)
	if !ok {
		panic("thread not found")
	}

	// tmp, ok := threadTraces.Load(threadID)
	// trace := tmp.([]string)
	// trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	// f, ok := threadFiles.Load(threadID)
	// if ok {
	// 	writer := f.(*bufio.Writer)
	// 	writer.WriteString(fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "?", s, "P", "-"))
	// }

	tmp, ok := threadTraces.Load(threadID)
	if ok {
		trace := tmp.([]string)
		trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "?", s, "P", "-"))
		threadTraces.Store(threadID, trace)
	}
}

func PostRcv(x interface{}, s string, p uint64) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name, ok := chans.Load(addr)

		if !ok {
			panic("chan not found")
		}

		chanID = name.(string)
	} else {
		chanID = "0,0"
	}

	threadID, ok := threads.Load(thread)
	if !ok {
		panic("thread not found")
	}
	pID, ok := threads.Load(p)
	if !ok {
		panic("partner thread not found")
	}

	// tmp, ok := threadTraces.Load(threadID)
	// trace := tmp.([]string)
	// trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	// f, ok := threadFiles.Load(threadID)
	// if ok {
	// 	writer := f.(*bufio.Writer)
	// 	writer.WriteString(fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "?", s, "C", pID))
	// }

	tmp, ok := threadTraces.Load(threadID)
	if ok {
		trace := tmp.([]string)
		trace = append(trace, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v\n", threadID, chanID, "?", s, "C", pID))
		threadTraces.Store(threadID, trace)
	}
}

func RegisterChan(x interface{}, c int) {
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	id := atomic.AddUint64(&chanID, 1)
	chans.Store(addr, fmt.Sprintf("%v,%v", id, c))
}

func RegisterThread(s string, id uint64) {
	thread := GetGID()
	threads.Store(thread, threadID)
	tmp := fmt.Sprintf("%v,[(%v,0,W,-)],C,-", threadID, id)
	threadTraces.Store(threadID, []string{tmp})

	// f, err := os.Create(fmt.Sprintf("%v.log", threadID))
	// if err != nil {
	// 	panic(err)
	// }
	// w := bufio.NewWriter(f)
	// //w.WriteString(fmt.Sprintf("%v,[(%v,0,W,-)],C,-\n", threadID, id))
	// threadFiles.Store(threadID, w)

	atomic.AddUint64(&threadID, 1)

	atomic.AddUint64(&threadsExisting, 1)
}

func StopThread() {
	atomic.AddUint64(&threadsExisting, ^uint64(0))
}

func AddSignal(id uint64) {
	thread := GetGID()
	threadID, ok := threads.Load(thread)

	if !ok {
		panic("thread not found!")
	}

	// tmp, ok := threadFiles.Load(threadID)
	// f := tmp.(*bufio.Writer)
	// f.WriteString(fmt.Sprintf("%v,[(%v,0,S,-)],C,-\n", threadID, id))
	tmp, ok := threadTraces.Load(threadID)
	if ok {
		trace := tmp.([]string)
		trace = append(trace, fmt.Sprintf("%v,[(%v,0,S,-)],C,-\n", threadID, id))
		threadTraces.Store(threadID, trace)
	}
}

func Start() {

}

func Stop() {
	for {
		if atomic.LoadUint64(&threadsExisting) == 0 {
			return
		}
		//fmt.Println(atomic.LoadUint64(&threadsExisting))
		time.Sleep(1 * time.Second)
	}
}
