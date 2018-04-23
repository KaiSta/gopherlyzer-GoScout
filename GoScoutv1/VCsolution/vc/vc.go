package vc

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

import "./cmap"

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

var f *os.File
var fLock *sync.Mutex

func init() {
	f, _ = os.Create("./trace.log")
	fLock = &sync.Mutex{}

	watchDog = make(chan struct{})
	done = make(chan struct{})
	traces = hashmap.NewCMapv2()
}

type VectorClock map[uint64]int

func (vc VectorClock) String() string {
	s := "["
	for k, v := range vc {
		s += fmt.Sprintf("(%v,%v)", k, v)
	}
	s += "]"
	return s
}

func (vc VectorClock) clone() VectorClock {
	nvc := NewVC()
	for k, v := range vc {
		nvc[k] = v
	}
	return nvc
}

func NewVC() VectorClock {
	return make(VectorClock)
}

type ChanString struct {
	ch chan struct {
		m  string
		vc VectorClock
		x  chan VectorClock
	}
	name string
}

func NewChanString(name string) *ChanString {
	return &ChanString{ch: make(chan struct {
		m  string
		vc VectorClock
		x  chan VectorClock
	}, 0), name: name}
}

func (ch *ChanString) Send(m string, vc VectorClock) {
	tid := GetGID()
	tmp := vc[tid]
	tmp++
	vc[tid] = tmp

	ret := make(chan VectorClock)
	ch.ch <- struct {
		m  string
		vc VectorClock
		x  chan VectorClock
	}{m, vc.clone(), ret}
	pvc := <-ret

	//sync vclocks
	for k, v := range vc {
		pv := pvc[k]
		vc[k] = max(v, pv)
	}
	for k, v := range pvc {
		pv := vc[k]
		vc[k] = max(pv, v)
	}

	traces.Store(tid, fmt.Sprintf("%v,%v,%v,-,%v", tid, ch.name, "!", vc))
	informWatchDog()
	// fLock.Lock()
	// defer fLock.Unlock()
	// f.WriteString(fmt.Sprintf("%v,%v,%v,-,%v\n", tid, ch.name, "!", vc))
}

func (ch *ChanString) Rcv(vc VectorClock) string {
	tid := GetGID()
	tmp := vc[tid]
	tmp++
	vc[tid] = tmp

	m := <-ch.ch
	m.x <- vc.clone()

	//sync vclocks
	for k, v := range vc {
		pv := m.vc[k]
		vc[k] = max(v, pv)
	}
	for k, v := range m.vc {
		pv := vc[k]
		vc[k] = max(pv, v)
	}

	traces.Store(tid, fmt.Sprintf("%v,%v,%v,-,%v", tid, ch.name, "?", vc))
	informWatchDog()
	// fLock.Lock()
	// defer fLock.Unlock()
	// f.WriteString(fmt.Sprintf("%v,%v,%v,-,%v\n", tid, ch.name, "?", vc))

	return m.m
}

type ChanInt struct {
	ch chan struct {
		m  int
		vc VectorClock
		x  chan VectorClock
	}
	name string
}

func NewChanInt(name string) *ChanInt {
	return &ChanInt{ch: make(chan struct {
		m  int
		vc VectorClock
		x  chan VectorClock
	}, 0), name: name}
}

func (ch *ChanInt) Send(m int, vc VectorClock) {
	tid := GetGID()
	tmp := vc[tid]
	tmp++
	vc[tid] = tmp

	ret := make(chan VectorClock)
	ch.ch <- struct {
		m  int
		vc VectorClock
		x  chan VectorClock
	}{m, vc.clone(), ret}
	pvc := <-ret

	//sync vclocks
	for k, v := range vc {
		pv := pvc[k]
		vc[k] = max(v, pv)
	}
	for k, v := range pvc {
		pv := vc[k]
		vc[k] = max(pv, v)
	}

	traces.Store(tid, fmt.Sprintf("%v,%v,%v,-,%v", tid, ch.name, "!", vc))
	informWatchDog()
	// fLock.Lock()
	// defer fLock.Unlock()
	// f.WriteString(fmt.Sprintf("%v,%v,%v,-,%v\n", tid, ch.name, "!", vc))
}

func (ch *ChanInt) Rcv(vc VectorClock) int {
	tid := GetGID()
	tmp := vc[tid]
	tmp++
	vc[tid] = tmp

	m := <-ch.ch
	m.x <- vc.clone()

	//sync vclocks
	for k, v := range vc {
		pv := m.vc[k]
		vc[k] = max(v, pv)
	}
	for k, v := range m.vc {
		pv := vc[k]
		vc[k] = max(pv, v)
	}

	traces.Store(tid, fmt.Sprintf("%v,%v,%v,-,%v", tid, ch.name, "?", vc))
	informWatchDog()
	//f.WriteString(fmt.Sprintf("%v,%v,%v,-,%v\n", tid, ch.name, "?", vc))

	return m.m
}

var watchDog chan struct{}
var done chan struct{}
var traces *hashmap.CMapv2

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
