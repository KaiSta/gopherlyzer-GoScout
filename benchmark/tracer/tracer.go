package tracer

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

type concSlice struct {
	sync.Mutex
	data []string
}
type rwmap struct {
	sync.RWMutex
	data map[uint64]*concSlice
}

var traces = rwmap{data: make(map[uint64]*concSlice)}
var threads = rwmap{data: make(map[uint64]*concSlice)}
var chans = rwmap{data: make(map[uint64]*concSlice)}

//counter to create unique ids
var chanID uint64
var threadID uint64
var waitSigID uint64

var watchDog = make(chan struct{})
var done = make(chan struct{})

func init() {
	arr := threads.data[1]
	if arr == nil {
		arr = &concSlice{}
	}
	arr.data = append(arr.data, "main")
	threads.data[1] = arr
	threadID = 2

	done = make(chan struct{})
	waitSigID = 1000
}

func GetWaitSigID() uint64 {
	return atomic.AddUint64(&waitSigID, 1)
}
func GetNewThreadID() uint64 {
	return atomic.AddUint64(&threadID, 1)
}
func GetNewChanID() uint64 {
	return atomic.AddUint64(&chanID, 1)
}

func informWatchDog() {
	go func() {
		watchDog <- struct{}{}
	}()
}

func storeInTraces(thread uint64, values ...string) {
	traces.RLock()
	vec, _ := traces.data[thread]
	traces.RUnlock()

	if vec == nil {
		vec = &concSlice{data: make([]string, 0)}
	}

	vec.Lock()
	for _, n := range values {
		vec.data = append(vec.data, n)
	}
	vec.Unlock()

	traces.Lock()
	traces.data[thread] = vec
	traces.Unlock()
}

func getThreadName(thread uint64) string {
	threads.RLock()
	arr, ok := threads.data[thread]
	threads.RUnlock()
	if !ok {
		panic("getThreadName couldn't find the thread identifier")
	}
	return arr.data[len(arr.data)-1]
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
	fileBuf.Flush()
}

var logfile *os.File
var fileBuf *bufio.Writer

func writeBack() {
	if fileBuf == nil {
		f, err := os.Create("./trace.log")
		if err != nil {
			panic(err)
		}
		fileBuf = bufio.NewWriter(f)
	}

	traces.Lock()
	for _, v := range traces.data {
		v.Lock()
		for i := range v.data {
			//logfile.WriteString(v.data[i] + "\n")
			fileBuf.WriteString(v.data[i] + "\n")
		}
		v.data = make([]string, 0)
		v.Unlock()
	}
	traces.Unlock()
	fileBuf.Flush()
}

func RegisterChan(x interface{}, c int) {
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	id := GetNewChanID()

	chans.RLock()
	arr, _ := chans.data[addr]
	chans.RUnlock()

	if arr == nil {
		arr = &concSlice{data: make([]string, 0)}
	}
	arr.Lock()
	arr.data = append(arr.data, fmt.Sprintf("%v,%v", id, c))
	arr.Unlock()

	chans.Lock()
	chans.data[addr] = arr
	chans.Unlock()
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func RegisterThread(s string) uint64 {
	//thread := GetNewThreadID()
	thread := GetGID()

	threadName := fmt.Sprintf("%v%v", s, thread)

	threads.RLock()
	arr, _ := threads.data[thread]
	threads.RUnlock()

	if arr == nil {
		arr = &concSlice{data: make([]string, 0)}
	}

	arr.Lock()
	arr.data = append(arr.data, threadName)
	arr.Unlock()

	threads.Lock()
	threads.data[thread] = arr
	threads.Unlock()

	return thread
}

func Wait(id uint64, thread uint64) {
	threads.RLock()
	arr, _ := threads.data[thread]
	threads.RUnlock()

	threadN := arr.data[len(arr.data)-1]

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,W,-)],C,-", threadN, id))

	informWatchDog()
}

func Signal(id uint64, thread uint64) {
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,S,-)],C,-", threadID, id))

	informWatchDog()
}

func getAddrAsString(x interface{}) string {
	if x != nil {
		v := reflect.ValueOf(x)
		vv := v.Elem()
		var addr uint64
		if vv.Kind() == reflect.Ptr {
			addr = uint64(vv.Pointer())
		} //else {
		addr = uint64(v.Pointer())
		//}
		return fmt.Sprint(addr)
	}
	return "nil"
}

func WriteAcc(x interface{}, s string, thread uint64) {
	name := getAddrAsString(x)

	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "M", s, "C", "-"))

	informWatchDog()
}
func ReadAcc(x interface{}, s string, thread uint64) {
	name := getAddrAsString(x)
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "R", s, "C", "-"))

	informWatchDog()
}

func handleLock(m interface{}, s string, thread uint64, capp uint64, op string) {
	// var name string
	// if m != nil {
	// 	v := reflect.ValueOf(m)
	// 	addr := uint64(v.Pointer())
	// 	name = fmt.Sprint(addr)
	// } else {
	// 	name = "nil"
	// }
	// v := reflect.ValueOf(m)
	// addr := uint64(v.Pointer())
	name := getAddrAsString(m)
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v,%v)],%v,%v", threadID, name, capp, op, s, "P", "-"),
		fmt.Sprintf("%v,[(%v,%v,%v,%v)],%v,%v", threadID, name, capp, op, s, "C", "-"))

	informWatchDog()
}

func PreLock(m interface{}, s string, thread uint64) {
	handleLock(m, s, thread, 1, "+")
}
func PostLock(m interface{}, s string, thread uint64) {
	handleLock(m, s, thread, 1, "*")
}

func RPreLock(m interface{}, s string, thread uint64) {
	handleLock(m, s, thread, 1, "$")
}
func RPostLock(m interface{}, s string, thread uint64) {
	handleLock(m, s, thread, 1, "#")
}

//deprecated. Normal mutexes are simulated with a async channel where PreLock is the async send on a channel size 1 and Postlock is the receive.
func PostUnlock(m interface{}, s string, thread uint64) {
	name := getAddrAsString(m)
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "P", "-"),
		fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "P", "-"),
		fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", threadID, name, "#", s, "C", "-"),
		fmt.Sprintf("%v,[(%v,0,%v,%v)],%v,%v", name, name, "*", s, "C", threadID))

	informWatchDog()
}

func getChanID(x interface{}) string {
	if x != nil {
		v := reflect.ValueOf(x)
		addr := uint64(v.Pointer())

		chans.RLock()
		arr, ok := chans.data[addr]
		chans.RUnlock()

		if !ok {
			panic("UNKNOWN CHAN. PreSend couldn't find channel id.")
		}

		return arr.data[len(arr.data)-1]
	}
	return "0,0"
}

func handleChanOp(x interface{}, s string, thread uint64, op, stat, partner string) {
	var chanID = getChanID(x)
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, op, s, stat, partner))

	informWatchDog()
}

func PreSend(x interface{}, s string, thread uint64) {
	handleChanOp(x, s, thread, "!", "P", "-")
}

func PostSend(x interface{}, s string, thread uint64) {
	handleChanOp(x, s, thread, "!", "C", "-")
}

func PreRcv(x interface{}, s string, thread uint64) {
	handleChanOp(x, s, thread, "?", "P", "-")
}

func PostRcv(x interface{}, s string, partner, thread uint64) {
	partnerID := "-"
	if partner != 0 {
		partnerID = getThreadName(partner)
	}
	handleChanOp(x, s, thread, "?", "C", partnerID)
}

type SelectEv struct {
	X  interface{}
	Op string
	S  string
}

func PreSelect(thread uint64, evs ...SelectEv) {
	chanevs := ""
	for i, x := range evs {
		var chanID = getChanID(x.X)
		chanevs += fmt.Sprintf("(%v,%v,%v)", chanID, x.Op, x.S)

		if i < len(evs)-1 {
			chanevs += ","
		}
	}
	threadID := getThreadName(thread)

	storeInTraces(thread, fmt.Sprintf("%v,[%v],%v,%v", threadID, chanevs, "P", 0))

	informWatchDog()
}

func PreClose(x interface{}, s string, thread uint64) {
	handleChanOp(x, s, thread, "C", "P", "-")
}
func PostClose(x interface{}, s string, thread uint64) {
	handleChanOp(x, s, thread, "C", "C", "-")
}
