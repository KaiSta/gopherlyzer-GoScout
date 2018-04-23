package vector

import (
	"sync/atomic"
	"unsafe"
)

const (
	defaultSize = 4
)

type CVector struct {
	root unsafe.Pointer
}

type entry struct {
	occ    uint64
	value  string
	remove unsafe.Pointer
}

type table struct {
	entries  []entry
	size     uint64
	cap      uint64
	jobSched unsafe.Pointer
}

type jobscheduler struct {
	newTable   *table // replaced the root table in CMap after the migration
	jobSize    uint64 // how many entries the old map has
	currStep   uint64 // idx of entry that is currently migrated by an arbitrary thread
	numThreads uint64 // number of threads that participate in the migration process
}

func newJobScheduler(jobSize uint64, t *table) *jobscheduler {
	return &jobscheduler{t, jobSize, 0, 0}
}

func (js *jobscheduler) participate() {
	atomic.AddUint64(&js.numThreads, 1)
}

const (
	newjob  = iota
	nojob   = iota
	publish = iota
	tba     = iota
)

type jobType int

func (j jobType) String() string {
	if j == newjob {
		return "newjob"
	} else if j == nojob {
		return "nojob"
	} else if j == publish {
		return "publish"
	}
	return "error"
}

func (js *jobscheduler) next() (uint64, jobType) {
	var cstep uint64
	var jt jobType = tba
	for jt == tba { // as long as we dont know what the next job will be
		cstep = atomic.LoadUint64(&js.currStep)
		if cstep < js.jobSize { // if there are still entries that need to be moved to the new map
			// cas is important, currstep could have been changed between the if condition and the add.
			if atomic.CompareAndSwapUint64(&js.currStep, cstep, cstep+1) {
				jt = newjob
			}
		} else {
			jt = nojob
		}
	} //repeat if there are still jobs left and we weren't able to aquire a next idx

	if jt == nojob {
		// check if this is the last thread. the last thread needs to publish the new map, the other threads move on and do nothing
		rThreads := atomic.AddUint64(&js.numThreads, ^uint64(0))
		if rThreads == 0 {
			jt = publish
		}
		/*
			if a thread enters this method after a thread received the command to publish the map, it returns nojob.
			This is because of rThreads := atomic.AddUint64(&js.numThreads, ^uint64(0)) which will be 0 for the last thread
			and for the following it will be max_uint64 - x. The jobtype stays at 'nojob' this way.
		*/
	}

	return cstep, jt
}

func NewCVector() *CVector {
	t := &table{entries: make([]entry, defaultSize), cap: defaultSize}
	return &CVector{unsafe.Pointer(t)}
}

func (v *CVector) Push_Back(s string) {
	for {
		t := (*table)(atomic.LoadPointer(&v.root))
		for i := t.size; i < t.cap; i++ {
			if atomic.CompareAndSwapUint64(&t.entries[i].occ, 0, 1) {
				t.entries[i].value = s
				atomic.AddUint64(&t.size, 1)
				return
			}
		}
		//no free fields, resize necessary
		v.resize(t)
	}
}

func (v *CVector) Get(idx int) string {
	t := (*table)(atomic.LoadPointer(&v.root))
	return t.entries[idx].value
}

func (v *CVector) Len() uint64 {
	t := (*table)(atomic.LoadPointer(&v.root))
	return atomic.LoadUint64(&t.size)
}

func (v *CVector) resize(t *table) {
	newT := &table{entries: make([]entry, t.cap*2), cap: t.cap * 2}
	js := newJobScheduler(t.cap, newT)

	atomic.CompareAndSwapPointer(&t.jobSched, nil, unsafe.Pointer(js))

	usedJs := (*jobscheduler)(atomic.LoadPointer(&t.jobSched))
	usedJs.participate()
	done := false
	for !done {
		idx, jt := usedJs.next()
		switch jt {
		case newjob:
			if atomic.LoadPointer(&t.entries[idx].remove) == nil {
				for i := uint64(0); i < usedJs.newTable.cap; i++ {
					if atomic.CompareAndSwapUint64(&usedJs.newTable.entries[i].occ, 0, 1) {
						usedJs.newTable.entries[i].value = t.entries[idx].value
						atomic.AddUint64(&usedJs.newTable.size, 1)
						break
					}
				}
			}
		case nojob:
			done = true
		case publish:
			atomic.SwapPointer(&v.root, unsafe.Pointer(usedJs.newTable))
		}
	}
}

type Iter struct {
	t    *table
	size uint64
	pos  uint64
}

func (i *Iter) HasNext() bool {
	// for j := i.pos + 1; j < i.size; j++ {
	// 	if atomic.LoadPointer(&i.t.entries[j].remove) == nil {
	// 		return true
	// 	}
	// }
	tmp := i.pos
	for tmp < i.t.size && atomic.LoadPointer(&i.t.entries[tmp].remove) != nil {
		tmp++
	}
	if tmp < i.size {
		return true
	}
	return false
}
func (i *Iter) Next() {
	i.pos++
	for i.pos < i.size && atomic.LoadPointer(&i.t.entries[i.pos].remove) != nil {
		i.pos++
	}
}
func (i *Iter) Get() string {
	return i.t.entries[i.pos].value
}
func (i *Iter) Delete() {
	atomic.CompareAndSwapPointer(&i.t.entries[i.pos].remove, nil, unsafe.Pointer(&struct{}{})) //abab delete during resize. migrated field i.pos already, so our delete is forgotten, change against flatcopy
}

func (v *CVector) Iterator() *Iter {
	t := (*table)(atomic.LoadPointer(&v.root))
	s := atomic.LoadUint64(&t.size)
	i := &Iter{t, s, 0}
	for atomic.LoadPointer(&i.t.entries[i.pos].remove) != nil {
		i.pos++
	}
	return i
}
