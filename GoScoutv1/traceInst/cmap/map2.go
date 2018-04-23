package hashmap

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"../vector"
)

type item struct {
	key uint64
	vec *vector.CVector
}

type table2 struct {
	items          []item
	size           uint64
	cellsRemaining int64
	jobSched       unsafe.Pointer
}

type CMapv2 struct {
	root unsafe.Pointer
}

func NewCMapv2() *CMapv2 {
	t := &table2{items: make([]item, defaultSize), size: defaultSize, cellsRemaining: (defaultSize * 3) / 4}
	return &CMapv2{unsafe.Pointer(t)}
}

func (cm *CMapv2) Store(key uint64, s string) {
	for {
		t := (*table2)(atomic.LoadPointer(&cm.root))

		res, idx := cm.insertOrFind(key, t)

		switch res {
		case insertedNew:
			t.items[idx].vec = vector.NewCVector()
			t.items[idx].vec.Push_Back(s)
			return
		case alreadyFound:
			t.items[idx].vec.Push_Back(s)
			return
		case overflow:
			cm.resize(t)
		}
	}
}

func (cm *CMapv2) Get(key uint64) *vector.CVector {
	for {
		t := (*table2)(atomic.LoadPointer(&cm.root))
		res, idx := cm.insertOrFind(key, t)

		switch res {
		case alreadyFound:
			return t.items[idx].vec
		}
	}
}

func (cm *CMapv2) resize(t *table2) {
	//optimistically create a new jobscheduler and an empty table
	newT := &table2{items: make([]item, t.size*2), size: t.size * 2, cellsRemaining: (int64(t.size*2) * 3) / 4}
	js := newJobScheduler2(t.size, newT)

	//try to swap the new created js against the js of the table. If the table scheduler is nil then its the first thread that starts the migration,
	// if its not nil, participate in the on going migration.
	atomic.CompareAndSwapPointer(&t.jobSched, nil, unsafe.Pointer(js))

	usedJs := (*jobscheduler2)(atomic.LoadPointer(&t.jobSched))
	usedJs.participate()
	done := false
	for !done {
		idx, jt := usedJs.next()
		switch jt {
		case newjob:
			cm.migrate(&t.items[idx], usedJs.newTable)
		case nojob:
			done = true
		case publish: // last thread according to the job scheduler, publish the new created table
			atomic.SwapPointer(&cm.root, unsafe.Pointer(usedJs.newTable))
			done = true
		}
	}
}

func (cm *CMapv2) migrate(e *item, t *table2) {
	res, idx := cm.insertOrFind(e.key, t)

	switch res {
	case insertedNew:
		t.items[idx].vec = e.vec //flat copy so we don't lose updates that happen during the migration (value is a *Trace)
	case alreadyFound:
		// e= empty
	case overflow:
		fmt.Println("should not happen")
	}
}

func (cm *CMapv2) insertOrFind(key uint64, t *table2) (result, uint64) {
	for idx := hash_uint64(key); ; idx++ {
		idx &= t.size - 1

		probedKey := atomic.LoadUint64(&t.items[idx].key)

		if probedKey == key {
			//key found in table
			return alreadyFound, idx
		}
		if probedKey == 0 {
			// empty cell try to reserve it
			// first: ensure that we can take a new cell
			r := atomic.AddInt64(&t.cellsRemaining, -1)
			if r <= 0 {
				// table is full
				atomic.AddInt64(&t.cellsRemaining, 1)
				return overflow, idx
			}

			//reserve cell
			succ := atomic.CompareAndSwapUint64(&t.items[idx].key, 0, key)
			if succ {
				// reserved cell successfully
				return insertedNew, idx
			}

			atomic.AddInt64(&t.cellsRemaining, 1)
			k := atomic.LoadUint64(&t.items[idx].key)
			if k == key {
				// other thread stored item with same hash
				return alreadyFound, idx
			}
		}
	}
}

type iter struct {
	t   *table2
	pos uint64
}

func (i *iter) HasNext() bool {
	for i.pos < i.t.size && i.t.items[i.pos].key == 0 {
		i.pos++
	}
	if i.pos < i.t.size {
		return true
	}
	return false
}
func (i *iter) Next() {
	i.pos++
}
func (i *iter) Get() *vector.Iter {
	return i.t.items[i.pos].vec.Iterator()
}

func (cm *CMapv2) Iterator() *iter {
	t := (*table2)(atomic.LoadPointer(&cm.root))
	return &iter{t, 0}
}

func (cm *CMapv2) Print() {
	t := (*table2)(atomic.LoadPointer(&cm.root))
	for _, e := range t.items {
		if e.key != 0 {
			fmt.Println("Key=", e.key)
			iter := e.vec.Iterator()
			for iter.HasNext() {
				fmt.Println(iter.Get())
				iter.Next()
			}
			fmt.Println("----")
		}
	}
}

type jobscheduler2 struct {
	newTable   *table2 // replaced the root table in CMap after the migration
	jobSize    uint64  // how many entries the old map has
	currStep   uint64  // idx of entry that is currently migrated by an arbitrary thread
	numThreads uint64  // number of threads that participate in the migration process
}

func newJobScheduler2(jobSize uint64, t *table2) *jobscheduler2 {
	return &jobscheduler2{t, jobSize, 0, 0}
}

func (js *jobscheduler2) participate() {
	atomic.AddUint64(&js.numThreads, 1)
}

func (js *jobscheduler2) next() (uint64, jobType) {
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
