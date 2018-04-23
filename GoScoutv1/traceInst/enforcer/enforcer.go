package enforcer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"../cmap"
)

var chans *hashmap.CMapv2
var traces *hashmap.CMapv2
var threads *hashmap.CMapv2
var chanID uint64
var watchDog chan struct{}
var done chan struct{}
var schedule []string
var con *sync.Cond

func init() {
	chans = hashmap.NewCMapv2()
	traces = hashmap.NewCMapv2()
	threads = hashmap.NewCMapv2()
	threads.Store(1, "main")
	watchDog = make(chan struct{})
	done = make(chan struct{})
	con = sync.NewCond(&sync.Mutex{})
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

const (
	PREPARE = 1 << iota
	COMMIT  = 1 << iota
	SEND    = 1 << iota
	RCV     = 1 << iota
)

type ev struct {
	threadID string
	chanID   string
	kind     int
	done     bool
}
type conditionEvent struct {
	events []ev
	con    *sync.Cond
}

var conEvents []conditionEvent

func prepare() {
	conEvents = make([]conditionEvent, 0)
	data, err := ioutil.ReadFile("schedule.txt")
	if err != nil {
		panic(err)
	}
	splitted := strings.Split(string(data), "\n")
	for _, s := range splitted {
		if len(s) <= 1 {
			break
		}
		csv := strings.Split(s, ",")

		conEv := conditionEvent{con: sync.NewCond(&sync.Mutex{})}
		op := SEND
		if csv[1] == "?" {
			op = RCV
		}
		conEv.events = append(conEv.events, ev{csv[0], csv[4], op | PREPARE, false})
		conEv.events = append(conEv.events, ev{csv[0], csv[4], op | COMMIT, false})

		op = SEND
		if csv[3] == "?" {
			op = RCV
		}
		conEv.events = append(conEv.events, ev{csv[2], csv[4], op | PREPARE, false})
		conEv.events = append(conEv.events, ev{csv[2], csv[4], op | COMMIT, false})

		conEvents = append(conEvents, conEv)
	}
	fmt.Println(conEvents)
}
func wait(chanID, threadID string, op int) {
	con.L.Lock()
	for len(conEvents) > 0 {
		conEv := conEvents[0]
		wait := true
		done := 0
		for i, e := range conEv.events {
			if !e.done && threadID == e.threadID && op == e.kind {
				conEv.events[i].done = true
				wait = false
			}
		}
		if wait {
			con.Wait()
		} else {
			for _, e := range conEv.events {
				if e.done {
					done++
				}
			}
			if done == len(conEv.events) {
				if len(conEvents) == 1 {
					conEvents = []conditionEvent{}
				} else {
					conEvents = conEvents[1:]
				}
				con.Broadcast()
			}
			con.L.Unlock()
			return
		}
	}
	con.L.Unlock()
}

func Start() {
	prepare()
	go func() { //watchdog
		for {
			select {
			case <-watchDog:
			case <-done:
				fmt.Println("DONE")

				writeBack()
				done <- struct{}{}
			case <-time.After(1 * time.Second):
				fmt.Println("Timeout")
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

func RegisterChan(x interface{}) {
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	id := atomic.AddUint64(&chanID, 1)
	chans.Store(addr, fmt.Sprintf("%v", id))
}

func RegisterThread(s string) {
	thread := GetGID()
	threads.Store(thread, s)
}

func SendPrep(x interface{}, s string) {
	thread := GetGID()
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	vec := chans.Get(addr)
	var chanID string
	iter := vec.Iterator()
	for iter.HasNext() {
		chanID = iter.Get()
		iter.Next()
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
	wait(chanID, threadID, SEND|PREPARE)
	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "P", "-"))

	informWatchDog()
}

func SendCommit(x interface{}, s string) {
	thread := GetGID()
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	vec := chans.Get(addr)
	var chanID string
	iter := vec.Iterator()
	for iter.HasNext() {
		chanID = iter.Get()
		iter.Next()
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
	wait(chanID, threadID, SEND|COMMIT)
	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "!", s, "C", "-"))

	informWatchDog()
}

func RcvPrep(x interface{}, s string) {
	thread := GetGID()
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	vec := chans.Get(addr)
	var chanID string
	iter := vec.Iterator()
	for iter.HasNext() {
		chanID = iter.Get()
		iter.Next()
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
	wait(chanID, threadID, RCV|PREPARE)
	//threadid,chanid,op,location,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[(%v,%v,%v)],%v,%v", threadID, chanID, "?", s, "P", "-"))

	informWatchDog()
}

func RcvCommit(x interface{}, s string, p uint64) {
	thread := GetGID()
	v := reflect.ValueOf(x)
	addr := uint64(v.Pointer())
	vec := chans.Get(addr)
	var chanID string
	iter := vec.Iterator()
	for iter.HasNext() {
		chanID = iter.Get()
		iter.Next()
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

	wait(chanID, threadID, RCV|COMMIT)

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
		v := reflect.ValueOf(x.X)
		addr := uint64(v.Pointer())
		vec := chans.Get(addr)
		var chanID string
		iter := vec.Iterator()
		for iter.HasNext() {
			chanID = iter.Get()
			iter.Next()
		}
		chanevs += fmt.Sprintf("(%v,%v,%v)", chanID, x.Op, x.S)

		if i < len(evs)-1 {
			chanevs += ","
		}
	}
	//threadid,chanops,status,partner
	traces.Store(thread, fmt.Sprintf("%v,[%v],%v,%v", thread, chanevs, "P", 0))

	informWatchDog()
}
