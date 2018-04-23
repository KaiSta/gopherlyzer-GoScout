package hashmap

// import (
// 	"fmt"
// 	"sync/atomic"
// 	"unsafe"
// )

// /*
// comments:
// simplify migration by using string pointers. if we copy the string pointer during migration we won't loose updates. since only one thread will manipulate the string its ok to leave it unprotected!
// */

const (
	defaultSize = 2
)

// type CMap struct {
// 	root unsafe.Pointer
// }

// type table struct {
// 	entries        []entry
// 	size           uint64
// 	cellsRemaining int64
// 	jobSched       unsafe.Pointer

// 	// only for multithreaded migration - replaced by a job scheduler or optimistic resizing
// 	//mutex          sync.Mutex
// 	//	newTable *table
// }

// type entry struct {
// 	key   uint64
// 	value *Trace
// }

// type Trace struct {
// 	Content string
// }

// type jobscheduler struct {
// 	newTable   *table // replaced the root table in CMap after the migration
// 	jobSize    uint64 // how many entries the old map has
// 	currStep   uint64 // idx of entry that is currently migrated by an arbitrary thread
// 	numThreads uint64 // number of threads that participate in the migration process
// }

// func newJobScheduler(jobSize uint64, t *table) *jobscheduler {
// 	return &jobscheduler{t, jobSize, 0, 0}
// }

// func (js *jobscheduler) participate() {
// 	atomic.AddUint64(&js.numThreads, 1)
// }

const (
	newjob  = iota
	nojob   = iota
	publish = iota
	tba     = iota
)

type jobType int

// func (j jobType) String() string {
// 	if j == newjob {
// 		return "newjob"
// 	} else if j == nojob {
// 		return "nojob"
// 	} else if j == publish {
// 		return "publish"
// 	}
// 	return "error"
// }

// func (js *jobscheduler) next() (uint64, jobType) {
// 	var cstep uint64
// 	var jt jobType = tba
// 	for jt == tba { // as long as we dont know what the next job will be
// 		cstep = atomic.LoadUint64(&js.currStep)
// 		if cstep < js.jobSize { // if there are still entries that need to be moved to the new map
// 			// cas is important, currstep could have been changed between the if condition and the add.
// 			if atomic.CompareAndSwapUint64(&js.currStep, cstep, cstep+1) {
// 				jt = newjob
// 			}
// 		} else {
// 			jt = nojob
// 		}
// 	} //repeat if there are still jobs left and we weren't able to aquire a next idx

// 	if jt == nojob {
// 		// check if this is the last thread. the last thread needs to publish the new map, the other threads move on and do nothing
// 		rThreads := atomic.AddUint64(&js.numThreads, ^uint64(0))
// 		if rThreads == 0 {
// 			jt = publish
// 		}
// 		/*
// 			if a thread enters this method after a thread received the command to publish the map, it returns nojob.
// 			This is because of rThreads := atomic.AddUint64(&js.numThreads, ^uint64(0)) which will be 0 for the last thread
// 			and for the following it will be max_uint64 - x. The jobtype stays at 'nojob' this way.
// 		*/
// 	}

// 	return cstep, jt
// }

// //murmurhash for 32bit
// func hash_uint32(h uint32) uint32 {
// 	h ^= h >> 16
// 	h *= 0x85ebca6b
// 	h ^= h >> 13
// 	h *= 0xc2b2ae35
// 	h ^= h >> 16

// 	return h
// }

func hash_uint64(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// func NewCMap() *CMap {
// 	t := &table{entries: make([]entry, defaultSize), size: defaultSize, cellsRemaining: (defaultSize * 3) / 4}
// 	return &CMap{unsafe.Pointer(t)}
// }

// func (cm *CMap) Get(key uint64) *Trace {
// 	for {
// 		t := (*table)(atomic.LoadPointer(&cm.root))

// 		res, idx := cm.insertOrFind(key, t)

// 		switch res {
// 		case insertedNew:
// 			t.entries[idx].value = &Trace{}
// 			return t.entries[idx].value
// 		case alreadyFound:
// 			return t.entries[idx].value
// 		case overflow:
// 			cm.resize2(t)
// 		}
// 	}
// }

// func (cm *CMap) resize2(t *table) {
// 	// /* potential bug here:
// 	// a thread becomes the nojob signal from the scheduler and returns to the resize function. loading the table pointer there, it encounters the old table that needs resizing
// 	// and calls the resize function again, if the table pointer was swapped in between the loading of the pointer here will return the new table which is then resized again unnecessarily.
// 	// the table pointer loaded in the insert function needs to be used here instead to ensure that it still uses the old table when it enters the resize function which will then contain the old
// 	// jobscheduler which sends the nojob signal again.
// 	// */
// 	// oldRoot := atomic.LoadPointer(&cm.root)
// 	// t := (*table)(oldRoot)

// 	//optimistically create a new jobscheduler and an empty table
// 	newT := &table{entries: make([]entry, t.size*2), size: t.size * 2, cellsRemaining: (int64(t.size*2) * 3) / 4}
// 	js := newJobScheduler(t.size, newT)

// 	//try to swap the new created js against the js of the table. If the table scheduler is nil then its the first thread that starts the migration,
// 	// if its not nil, participate in the on going migration.
// 	atomic.CompareAndSwapPointer(&t.jobSched, nil, unsafe.Pointer(js))

// 	usedJs := (*jobscheduler)(atomic.LoadPointer(&t.jobSched))
// 	usedJs.participate()
// 	done := false
// 	for !done {
// 		idx, jt := usedJs.next()
// 		switch jt {
// 		case newjob:
// 			cm.migrate(&t.entries[idx], usedJs.newTable)
// 		case nojob:
// 			done = true
// 		case publish: // last thread according to the job scheduler, publish the new created table
// 			atomic.SwapPointer(&cm.root, unsafe.Pointer(usedJs.newTable))
// 			done = true
// 		}
// 	}
// }

// // old version using a lock to prevent multiple threads to create a new table.
// // func (cm *CMap) resize() {
// // 	oldRoot := atomic.LoadPointer(&cm.root)
// // 	t := (*table)(oldRoot)
// // 	t.mutex.Lock()
// // 	if t.newTable != nil {
// // 		t.mutex.Unlock()
// // 		return
// // 	}

// // 	newT := &table{entries: make([]entry, t.size*2), size: t.size * 2, cellsRemaining: (int32(t.size*2) * 3) / 4}
// // 	for i := range t.entries {
// // 		cm.migrate(&t.entries[i], newT)
// // 	}
// // 	t.newTable = newT
// // 	succ := atomic.CompareAndSwapPointer(&cm.root, oldRoot, unsafe.Pointer(newT))
// // 	if succ {
// // 		t.mutex.Unlock()
// // 		return
// // 	}
// // }

// func (cm *CMap) migrate(e *entry, t *table) {
// 	res, idx := cm.insertOrFind(e.key, t)

// 	switch res {
// 	case insertedNew:
// 		t.entries[idx].value = e.value //flat copy so we don't lose updates that happen during the migration (value is a *Trace)
// 	case alreadyFound:
// 		// e= empty
// 	case overflow:
// 		fmt.Println("should not happen")
// 	}
// }

// func (cm *CMap) PrintMap() {
// 	t := (*table)(atomic.LoadPointer(&cm.root))
// 	fmt.Printf("Size=%v, free=%v, [", t.size, t.cellsRemaining)
// 	for i := range t.entries {
// 		if t.entries[i].key != 0 {
// 			val := t.entries[i].value
// 			fmt.Printf("(%v,%v)--", t.entries[i].key, *val)
// 		}

// 	}
// 	fmt.Print("]\n")
// }

const (
	insertedNew  = iota
	alreadyFound = iota
	overflow     = iota
)

type result int

// func (cm *CMap) insertOrFind(key uint64, t *table) (result, uint64) {
// 	for idx := hash_uint64(key); ; idx++ {
// 		idx &= t.size - 1

// 		probedKey := atomic.LoadUint64(&t.entries[idx].key)

// 		if probedKey == key {
// 			//key found in table
// 			return alreadyFound, idx
// 		}
// 		if probedKey == 0 {
// 			// empty cell try to reserve it
// 			// first: ensure that we can take a new cell
// 			r := atomic.AddInt64(&t.cellsRemaining, -1)
// 			if r <= 0 {
// 				// table is full
// 				atomic.AddInt64(&t.cellsRemaining, 1)
// 				return overflow, idx
// 			}

// 			//reserve cell
// 			succ := atomic.CompareAndSwapUint64(&t.entries[idx].key, 0, key)
// 			if succ {
// 				// reserved cell successfully
// 				return insertedNew, idx
// 			}

// 			atomic.AddInt64(&t.cellsRemaining, 1)
// 			k := atomic.LoadUint64(&t.entries[idx].key)
// 			if k == key {
// 				// other thread stored item with same hash
// 				return alreadyFound, idx
// 			}
// 		}
// 	}
// }

// // func (cm *CMap) set(key uint32, val string, t *table) {
// // 		t := (*table)(atomic.LoadPointer(&cm.root))
// // 	rounds := uint32(0)
// // 	consecutive := false
// // 	for idx := hash_uint32(key); ; idx++ {
// // 		idx &= t.size - 1
// // 		k := atomic.LoadUint32(&t.entries[idx].key)

// // 		key-value already in map?
// // 		if k == key {
// // 			cm.updateEntry(idx, val, t)
// // 			return
// // 		}
// // 		new key-value and we have found an empty cell?
// // 		if k == 0 {
// // 			if consecutive is false, its the first try to reserve one of the fields
// // 			if its true, then we already reserved one spot and we are searching for it atm.
// // 			if !consecutive {
// // 				r := atomic.AddInt32(&t.cellsRemaining, -1) // reserve one of the free spots
// // 					fmt.Println("remaining=", r, "t.size=", t.size, "key=", key, "val=", val)
// // 				if r <= 0 { // if r becomes <0 then there was no free spot left and we need to resize the map
// // 							fmt.Println("resize1")
// // 					cm.resize(key, val)
// // 					return
// // 				}
// // 			}
// // 			consecutive = true
// // 			succ := atomic.CompareAndSwapUint32(&t.entries[idx].key, 0, key) //we successfully reserved a free spot, and check the current location if its available
// // 			if succ {
// // 				cm.updateEntry(idx, val, t)
// // 				return
// // 			}
// // 		}
// // 	}
// // }

// // func (cm *CMap) resize(key uint32, val string) {
// // 	//	for {
// // 	oldRoot := atomic.LoadPointer(&cm.root)
// // 	t := (*table)(oldRoot)
// // 	t.mutex.Lock()

// // 	if t.newTable != nil {
// // 		fmt.Println("DIFF")
// // 		t.mutex.Unlock()
// // 		cm.Set(key, val)
// // 		return
// // 	}

// // 	newT := &table{entries: make([]entry, t.size*2), size: t.size * 2, cellsRemaining: (int32(t.size*2) * 3) / 4}
// // 	dummyString := ""
// // 	for i := range newT.entries {
// // 		newT.entries[i].value = unsafe.Pointer(&dummyString)
// // 	}
// // 	succ := atomic.CompareAndSwapPointer(&t.newTable, nil, unsafe.Pointer(newT))
// // 	if succ {
// // 		cm.set(key, val, newT)
// // 		for i := range t.entries {
// // 			//compareAndSwap for redirect
// // 			cm.migrate(&t.entries[i], newT)
// // 		}

// // 		succ2 := atomic.CompareAndSwapPointer(&cm.root, oldRoot, unsafe.Pointer(newT))
// // 		if succ2 {
// // 			t.mutex.Unlock()
// // 			return
// // 		}
// // 	}
// // 	t.mutex.Unlock()
// // 	//}
// // }

// // func (cm *CMap) migrate(e *entry, t *table) {
// // 	rounds := uint32(0)
// // 	for idx := hash_uint32(e.key); rounds < t.size; idx++ {
// // 		rounds++
// // 		idx &= t.size - 1
// // 		// search for a free space in the new table
// // 		// if e is an empty entry, we copy the 0-key from the old entry to the new one
// // 		// in both cases we need to set the value field to nil, to signal that the field migrated to a new table
// // 		succ := atomic.CompareAndSwapUint32(&t.entries[idx].key, 0, e.key)
// // 		if succ {
// // 			//we swap the value from the old entry with nil, to signal that it was migrated to a new table
// // 			for {
// // 				pString := atomic.LoadPointer(&e.value)
// // 				tmp := (*string)(pString)
// // 				t.entries[idx].value = unsafe.Pointer(tmp) // set the value of the new entry to the value of the old entry, before setting the redirect mark
// // 				// to ensure that a thread that follows the redirection won't find an empty value field here.
// // 				succ := atomic.CompareAndSwapPointer(&e.value, pString, nil)
// // 				if succ {
// // 					return
// // 				}
// // 			}
// // 		}
// // 	}
// // 	panic("migration error")
// // }

// // func (cm *CMap) updateEntry(idx uint32, val string, t *table) {
// // 	oldV := atomic.LoadPointer(&t.entries[idx].value)
// // 	if oldV == nil { //check if the value was already migrated to a new hashmap
// // 		nt := atomic.LoadPointer(&t.newTable)
// // 		cm.set(t.entries[idx].key, val, (*table)(nt)) //update the value in the new hashmap
// // 		return
// // 	}
// // 	succ := atomic.CompareAndSwapPointer(&t.entries[idx].value, oldV, unsafe.Pointer(&val))
// // 	if succ {
// // 		// either the updating thread won the race against the migration thread
// // 		// OR there is no on going migration
// // 		return
// // 	}
// // 	// the only competition for the above cas is a migration thread. The migration thread
// // 	// only writes nil into the entries value. Therefore if we lost the cas above, it
// // 	// means the current value of t.entries[idx].value is nil and already migrated
// // 	// to a new hashmap.
// // 	nt := atomic.LoadPointer(&t.newTable)
// // 	cm.set(t.entries[idx].key, val, (*table)(nt))
// // 	return
// // }

// // func (cm *CMap) get(key uint32, t *table) string {
// // 	rounds := uint32(0)
// // 	for idx := hash_uint32(key); rounds < t.size; idx++ {
// // 		idx &= t.size - 1
// // 		probe := atomic.LoadUint32(&t.entries[idx].key)
// // 		if probe == key {
// // 			v := atomic.LoadPointer(&t.entries[idx].value)
// // 			s := (*string)(v)
// // 			if s == nil {
// // 				nt := (*table)(atomic.LoadPointer(&t.newTable))
// // 				cm.get(key, nt)
// // 			}
// // 			return *(s)
// // 		}
// // 		if probe == 0 {
// // 			break
// // 		}
// // 	}
// // 	return ""
// // }
