package tracer3

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"../vector"
)

//var chans *hashmap.CMapv2
var chans sync.Map

//var traces *hashmap.CMapv2
var traces sync.Map

//var threads *hashmap.CMapv2
var threads sync.Map
var chanID uint64
var threadID uint64
var watchDog chan struct{}
var done chan struct{}
var waitSigID uint64

type concurrentSlice struct {
	sync.Mutex
	data []string
}

func init() {
	//chans = hashmap.NewCMapv2()
	chans = sync.Map{}
	//traces = hashmap.NewCMapv2()
	traces = sync.Map{}
	//threads = hashmap.NewCMapv2()
	threads = sync.Map{}

	threads.Store(uint64(1), uint64(1))
	threadID = 2

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

func GetWaitSigID() uint64 {
	return atomic.AddUint64(&waitSigID, 1)
}

func informWatchDog() {
	go func() {
		watchDog <- struct{}{}
	}()
}

func storeInTraces(thread uint64, values ...string) {
	vec, _ := traces.Load(thread)
	if vec == nil {
		vec = &concurrentSlice{data: make([]string, 0)} //vector.NewCVector()
	}

	s := vec.(*concurrentSlice)
	s.Lock()
	for _, n := range values {
		s.data = append(s.data, n)
		//	vec.(*vector.CVector).Push_Back(n)
	}
	s.Unlock()
	traces.Store(thread, s)
}

func getThreadName(thread uint64) string {
	threadID := "-"
	vec, ok := threads.Load(thread)
	// threads.Range(func(key, value interface{}) bool {
	// 	fmt.Println(key, value, "||", thread)
	// 	return true
	// })
	if !ok {
		threads.Range(func(key, value interface{}) bool {
			fmt.Println(key, value, "||", thread)
			return true
		})
		panic("something bad")
	}

	if vec != nil {
		iter2 := vec.(*vector.CVector).Iterator()
		for iter2.HasNext() {
			threadID = iter2.Get()
			iter2.Next()
		}
	}
	return threadID
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

var logfile *os.File

func writeBack() {
	if logfile == nil {
		f, err := os.Create("./trace.log")
		if err != nil {
			panic(err)
		}
		logfile = f
	}

	traces.Range(func(key, value interface{}) bool {
		vec := value.(*concurrentSlice)

		vec.Lock()
		for i := range vec.data {
			logfile.WriteString(vec.data[i] + "\n")
		}
		vec.data = make([]string, 0)
		vec.Unlock()

		return true
	})

	// iter := traces.Iterator()
	// for iter.HasNext() {
	// 	iter2 := iter.Get()
	// 	for iter2.HasNext() {
	// 		f.WriteString(iter2.Get() + "\n")

	// 		iter2.Next()
	// 	}
	// 	iter.Next()
	// }
}

func RegisterChan(x interface{}, c int) {
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	id := atomic.AddUint64(&chanID, 1)

	vec := vector.NewCVector()
	vec.Push_Back(fmt.Sprintf("%v,%v", id, c))
	chans.Store(addr, vec)
	//chans.Store(addr, fmt.Sprintf("%v,%v", id, c))
}

func RegisterThread(s string, id uint64) {
	thread := GetGID()

	threads.Store(thread, threadID)
	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,W,-)],C,-", threadID, id))
	atomic.AddUint64(&threadID, 1)

	informWatchDog()
}

func AddSignal(id uint64) {
	thread := GetGID()
	//	threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,S,-)],C,-", threadID, id))
	}

	informWatchDog()
}

func WriteAcc(x interface{}, s string) {
	thread := GetGID()
	var name string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name = fmt.Sprint(addr)
	} else {
		name = "nil"
	}

	//threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "M", s, "C", "-"))
	}
	informWatchDog()
}
func ReadAcc(x interface{}, s string) {
	thread := GetGID()
	var name string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		name = fmt.Sprint(addr)
	} else {
		name = "nil"
	}

	//threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "R", s, "C", "-"))
	}
	informWatchDog()
}

func PreLock(m interface{}, s string) {
	thread := GetGID()
	var name string

	if m != nil {
		v := reflect.ValueOf(m)
		addr := uint64(v.Pointer())
		name = fmt.Sprint(addr)
	} else {
		name = "nil"
	}

	//threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "+", s, "P", "-"))
	}
	informWatchDog()
}
func PostLock(m interface{}, s string) {
	thread := GetGID()
	var name string

	if m != nil {
		v := reflect.ValueOf(m)
		addr := uint64(v.Pointer())
		name = fmt.Sprint(addr)
	} else {
		name = "nil"
	}

	//	threadID := getThreadName(thread)

	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "-", s, "P", "-"),
			fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "+", s, "C", "-"),
			fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "-", s, "C", threadID))
	}
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "-", s, "P", "-"))
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "+", s, "C", "-"))
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "-", s, "C", threadID))
	informWatchDog()
}
func PostUnlock(m interface{}, s string) {
	thread := GetGID()
	var name string

	if m != nil {
		v := reflect.ValueOf(m)
		addr := uint64(v.Pointer())
		name = fmt.Sprint(addr)
	} else {
		name = "nil"
	}

	//threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "P", "-"),
			fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "P", "-"),
			fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "C", "-"),
			fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "C", threadID))
	}
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "P", "-"))
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "P", "-"))
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "C", "-"))
	// traces.Store(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "C", threadID))
	informWatchDog()
}

func PrepSend(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		//vec := chans.Get(addr)
		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//threadID := getThreadName(thread)

	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))
	}
	//threadid,chanid,op,location,status,partner
	//traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	informWatchDog()
}

func PostSend(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		//		vec := chans.Get(addr)
		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//threadID := getThreadName(thread)

	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "C", "-"))
	}
	//threadid,chanid,op,location,status,partner
	//	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "C", "-"))

	informWatchDog()
}

func PrepRcv(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		// vec := chans.Get(addr)

		// iter := vec.Iterator()
		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()

		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//	threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "P", "-"))
	}
	//threadid,chanid,op,location,status,partner
	//	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "P", "-"))

	informWatchDog()
}

func PostRcv(x interface{}, s string, p uint64) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		// vec := chans.Get(addr)

		// iter := vec.Iterator()

		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()

		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//	threadID := getThreadName(thread)

	//pID := getThreadName(p)

	//threadid,chanid,op,location,status,partner
	threadID, ok := threads.Load(thread)
	pID, ok2 := threads.Load(p)
	if ok && ok2 {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "C", pID))
	}
	//traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "C", pID))

	informWatchDog()
}

type SelectEv struct {
	X  interface{}
	Op string
	S  string
}

func PrepSelect(evs ...SelectEv) {
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
			// vec := chans.Get(addr)

			// iter := vec.Iterator()
			vec, ok := chans.Load(addr)
			if !ok {
				panic("Housten, we have a problem")
			}
			iter := vec.(*vector.CVector).Iterator()
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
	//threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		//threadid,chanops,status,partner
		storeInTraces(thread, fmt.Sprintf("%v,[%v],%v,%v", threadID, chanevs, "P", 0))
	}
	//traces.Store(thread, fmt.Sprintf("%v,[%v],%v,%v", threadID, chanevs, "P", 0))
	informWatchDog()
}

func PrepClose(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())

		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()
		// vec := chans.Get(addr)

		// iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//	threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "0", s, "P", "-"))
	}
	//threadid,chanid,op,location,status,partner
	//	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "#", s, "P", "-"))

	informWatchDog()
}
func PostClose(x interface{}, s string) {
	thread := GetGID()
	var chanID string

	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())
		vec, ok := chans.Load(addr)
		if !ok {
			panic("Housten, we have a problem")
		}
		iter := vec.(*vector.CVector).Iterator()
		// vec := chans.Get(addr)

		// iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
	} else {
		chanID = "0,0"
	}

	//	threadID := getThreadName(thread)
	threadID, ok := threads.Load(thread)

	if ok {
		storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "0", s, "C", "-"))
	}
	//threadid,chanid,op,location,status,partner
	//traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "#", s, "C", "-"))

	informWatchDog()
}
