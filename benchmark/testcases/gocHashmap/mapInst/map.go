package maptest

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"../../../tracer"
)

type Map struct {
	mu sync.Mutex

	read atomic.Value

	dirty map[interface{}]*entry

	misses int
}

type readOnly struct {
	m       map[interface{}]*entry
	amended bool
}

var expunged = unsafe.Pointer(new(interface{}))

type entry struct {
	p unsafe.Pointer
}

func newEntry(i interface{}) *entry {
	return &entry{p: unsafe.Pointer(&i)}
}

func (m *Map) Load(key interface{}) (value interface{}, ok bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:103:1", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:103:1", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:103:1", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:103:2", myTIDCache)

	e, ok := read.m[key]
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:104:1", myTIDCache)
	tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:104:2", myTIDCache)
	tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:104:3", myTIDCache)
	tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:105", myTIDCache)
	tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:105", myTIDCache)
	if !ok && read.amended {
		myTIDCache := tracer.GetGID()
		tracer.PreLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:106", myTIDCache)
		m.mu.Lock()
		tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:110", myTIDCache)
		read, _ = m.read.Load().(readOnly)
		tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:110", myTIDCache)
		tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:110", myTIDCache)
		tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:110", myTIDCache)
		e, ok = read.m[key]
		tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:111", myTIDCache)
		tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:111", myTIDCache)
		tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:111", myTIDCache)
		tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:112", myTIDCache)
		tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:112", myTIDCache)
		if !ok && read.amended {
			myTIDCache := tracer.GetGID()
			e, ok = m.dirty[key]
			tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:113", myTIDCache)
			tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:113", myTIDCache)
			tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:113", myTIDCache)

			m.missLocked()
		}
		tracer.PostLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:119", myTIDCache)
		m.mu.Unlock()
	}
	tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:121", myTIDCache)
	if !ok {
		return nil, false
	}
	tracer.PreLock(&e, "testcases\\gocHasmap\\map\\map.go.123", myTIDCache)
	tracer.ReadAcc(&e, "testcases\\gocHasmap\\map\\map.go.123", myTIDCache)
	tracer.PostLock(&e, "testcases\\gocHasmap\\map\\map.go.123", myTIDCache)
	return e.load()
}

func (e *entry) load() (value interface{}, ok bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:128", myTIDCache)
	tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:128", myTIDCache)
	p := atomic.LoadPointer(&e.p)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:128", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:128", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:129", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:129", myTIDCache)
	tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:129", myTIDCache)
	if p == nil || p == expunged {
		return nil, false
	}
	return *(*interface{})(p), true
}

func (m *Map) Store(key, value interface{}) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:137", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:137", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:137", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:137", myTIDCache)
	tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:138", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:138", myTIDCache)
	if e, ok := read.m[key]; ok && e.tryStore(&value) {
		return
	}
	tracer.PreLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:142", myTIDCache)
	m.mu.Lock()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:143", myTIDCache)
	read, _ = m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:143", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:143", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:143", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:144", myTIDCache)
	tracer.ReadAcc(&m.dirty, "testcases\\gocHasmap\\map\\map.go:151", myTIDCache)
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
			tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:148", myTIDCache)
			tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:148", myTIDCache)
		}
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:150", myTIDCache)
		e.storeLocked(&value)
	} else if e, ok := m.dirty[key]; ok {
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:152", myTIDCache)
		e.storeLocked(&value)
	} else {
		tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:154", myTIDCache)
		if !read.amended {
			m.dirtyLocked()
			tracer.PreLock(&m.read, "testcases\\gocHasmap\\map\\map.go:155", myTIDCache)
			m.read.Store(readOnly{m: read.m, amended: true})
			tracer.WriteAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:155", myTIDCache)
			tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:155", myTIDCache)
			tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:155", myTIDCache)
		}
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:160", myTIDCache)
		m.dirty[key] = newEntry(value)
		tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:160", myTIDCache)
	}
	tracer.PostLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:162", myTIDCache)
	m.mu.Unlock()
}

func (e *entry) tryStore(i *interface{}) bool {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:170", myTIDCache)
	tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:170", myTIDCache)
	p := atomic.LoadPointer(&e.p)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:170", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:170", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:171", myTIDCache)
	tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:171", myTIDCache)
	if p == expunged {
		return false
	}
	for {
		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:177", myTIDCache)
		if atomic.CompareAndSwapPointer(&e.p, p, unsafe.Pointer(i)) {
			return true
		}
		tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:177", myTIDCache)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:177", myTIDCache)
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:177", myTIDCache)

		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:178", myTIDCache)
		tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:178", myTIDCache)
		p = atomic.LoadPointer(&e.p)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:178", myTIDCache)
		tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:178", myTIDCache)
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:179", myTIDCache)
		tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:179", myTIDCache)
		if p == expunged {
			return false
		}
	}
}

func (e *entry) unexpungeLocked() (wasExpunged bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:180", myTIDCache)
	tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:180", myTIDCache)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:180", myTIDCache)
	tracer.ReadAcc(&expunged, "testcases\\gocHasmap\\map\\map.go:180", myTIDCache)
	return atomic.CompareAndSwapPointer(&e.p, expunged, nil)
}

func (e *entry) storeLocked(i *interface{}) {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&i, "testcases\\gocHashmap\\map\\map.go:197:1", myTIDCache)
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:197", myTIDCache)
	atomic.StorePointer(&e.p, unsafe.Pointer(i))
	tracer.WriteAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:197:2", myTIDCache)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:197", myTIDCache)
}

func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	myTIDCache := tracer.GetGID()

	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:205", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:205", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:205", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:205", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:206", myTIDCache)
	if e, ok := read.m[key]; ok {
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:207", myTIDCache)
		actual, loaded, ok := e.tryLoadOrStore(value)
		tracer.WriteAcc(&actual, "testcases\\gocHashmap\\map\\map.go:207", myTIDCache)
		tracer.WriteAcc(&loaded, "testcases\\gocHashmap\\map\\map.go:207", myTIDCache)
		tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:207", myTIDCache)
		if ok {
			return actual, loaded
		}
	}
	tracer.PreLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:213", myTIDCache)
	m.mu.Lock()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:214", myTIDCache)
	read, _ = m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:214", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:214", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:214", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:215", myTIDCache)
	tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:220", myTIDCache)
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
			tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:217", myTIDCache)
			tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:217", myTIDCache)
		}
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:219", myTIDCache)
		actual, loaded, _ = e.tryLoadOrStore(value)
		tracer.WriteAcc(&actual, "testcases\\gocHashmap\\map\\map.go:219", myTIDCache)
		tracer.WriteAcc(&loaded, "testcases\\gocHashmap\\map\\map.go:219", myTIDCache)
	} else if e, ok := m.dirty[key]; ok {
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:221", myTIDCache)
		actual, loaded, _ = e.tryLoadOrStore(value)
		tracer.WriteAcc(&actual, "testcases\\gocHashmap\\map\\map.go:221", myTIDCache)
		tracer.WriteAcc(&loaded, "testcases\\gocHashmap\\map\\map.go:221", myTIDCache)
		m.missLocked()
	} else {
		tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:224", myTIDCache)
		if !read.amended {
			m.dirtyLocked()
			tracer.PreLock(&m.read, "testcases\\gocHasmap\\map\\map.go:226", myTIDCache)
			tracer.WriteAcc(&m.read, "testcases\\gocHasmap\\map\\map.go:226", myTIDCache)
			tracer.PostLock(&m.read, "testcases\\gocHasmap\\map\\map.go:226", myTIDCache)
			m.read.Store(readOnly{m: read.m, amended: true})
			tracer.ReadAcc(&read.m, "testcases\\gocHasmap\\map\\map.go:226", myTIDCache)
		}
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:230", myTIDCache)
		m.dirty[key] = newEntry(value)
		tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:230", myTIDCache)
		actual, loaded = value, false
		tracer.ReadAcc(&value, "testcases\\gocHashmap\\map\\map.go:231", myTIDCache)
		tracer.WriteAcc(&actual, "testcases\\gocHashmap\\map\\map.go:231", myTIDCache)
		tracer.WriteAcc(&loaded, "testcases\\gocHashmap\\map\\map.go:231", myTIDCache)
	}
	tracer.PostLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:233", myTIDCache)
	m.mu.Unlock()

	return actual, loaded
}

func (e *entry) tryLoadOrStore(i interface{}) (actual interface{}, loaded, ok bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:244", myTIDCache)
	tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:244", myTIDCache)
	p := atomic.LoadPointer(&e.p)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:244", myTIDCache)
	tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:244", myTIDCache)
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:245", myTIDCache)
	tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:245", myTIDCache)
	if p == expunged {
		return nil, false, false
	}
	tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:248", myTIDCache)
	if p != nil {
		return *(*interface{})(p), true, true
	}

	ic := i
	tracer.ReadAcc(&i, "testcases\\gocHashmap\\map\\map.go:255", myTIDCache)
	tracer.WriteAcc(&ic, "testcases\\gocHashmap\\map\\map.go:255", myTIDCache)
	for {
		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:258", myTIDCache)
		tracer.WriteAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:258", myTIDCache)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:258", myTIDCache)
		if atomic.CompareAndSwapPointer(&e.p, nil, unsafe.Pointer(&ic)) {
			return i, false, true
		}
		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:260", myTIDCache)
		tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:260", myTIDCache)
		p = atomic.LoadPointer(&e.p)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:260", myTIDCache)
		tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:260", myTIDCache)
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:261", myTIDCache)
		tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:261", myTIDCache)
		if p == expunged {
			return nil, false, false
		}
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:264", myTIDCache)
		if p != nil {
			return *(*interface{})(p), true, true
		}
	}
}

func (m *Map) Delete(key interface{}) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:272", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:272", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:272", myTIDCache)

	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:272", myTIDCache)
	e, ok := read.m[key]
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:273", myTIDCache)
	tracer.WriteAcc(&e, "testcases\\gocHashmap\\map\\map.go:273", myTIDCache)
	tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:273", myTIDCache)
	tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:274", myTIDCache)
	tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:274", myTIDCache)
	if !ok && read.amended {
		tracer.PreLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:275", myTIDCache)
		m.mu.Lock()
		tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:276", myTIDCache)
		read, _ = m.read.Load().(readOnly)
		tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:276", myTIDCache)
		tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:276", myTIDCache)
		tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:276", myTIDCache)
		e, ok = read.m[key]
		tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:277", myTIDCache)
		tracer.WriteAcc(&e, "testcases\\gocHashmap\\map\\map.go:277", myTIDCache)
		tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:277", myTIDCache)
		tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:278", myTIDCache)
		tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:278", myTIDCache)
		if !ok && read.amended {
			tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:279", myTIDCache)
			tracer.ReadAcc(&key, "testcases\\gocHashmap\\map\\map.go:279", myTIDCache)
			delete(m.dirty, key)
			tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:279", myTIDCache)
		}
		tracer.PostLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:281", myTIDCache)
		m.mu.Unlock()
	}
	if ok {
		e.delete()
	}
}

func (e *entry) delete() (hadValue bool) {
	myTIDCache := tracer.GetGID()
	for {
		tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:290", myTIDCache)
		p := atomic.LoadPointer(&e.p)
		tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:290", myTIDCache)
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:291", myTIDCache)
		tracer.ReadAcc(&p, "testcases\\gocHashmap\\map\\map.go:291", myTIDCache)
		tracer.ReadAcc(&expunged, "testcases\\gocHashmap\\map\\map.go:291", myTIDCache)
		if p == nil || p == expunged {

			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, nil) {
			return true
		}
	}
}

func (m *Map) Range(f func(key, value interface{}) bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:315", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:315", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:315", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:315", myTIDCache)
	tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:316", myTIDCache)
	if read.amended {
		tracer.PreLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:321", myTIDCache)
		m.mu.Lock()
		tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:322", myTIDCache)
		read, _ = m.read.Load().(readOnly)
		tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:322", myTIDCache)
		tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:322", myTIDCache)
		tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:322", myTIDCache)
		tracer.ReadAcc(&read.amended, "testcases\\gocHashmap\\map\\map.go:323", myTIDCache)
		if read.amended {
			tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:324", myTIDCache)
			read = readOnly{m: m.dirty}
			tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:324", myTIDCache)
			tracer.ReadAcc(&read, "testcases\\gocHashmap\\map\\map.go:324", myTIDCache)
			tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:325", myTIDCache)
			m.read.Store(read)
			tracer.WriteAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:325", myTIDCache)
			tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:325", myTIDCache)
			m.dirty = nil
			tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:326", myTIDCache)
			m.misses = 0
			tracer.WriteAcc(&m.misses, "testcases\\gocHashmap\\map\\map.go:327", myTIDCache)
		}
		tracer.PostLock(&m.mu, "testcases\\gocHashmap\\map\\map.go:329", myTIDCache)
		m.mu.Unlock()
	}

	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:332", myTIDCache)
	for k, e := range read.m {
		tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:333", myTIDCache)
		v, ok := e.load()
		tracer.WriteAcc(&v, "testcases\\gocHashmap\\map\\map.go:333", myTIDCache)
		tracer.WriteAcc(&ok, "testcases\\gocHashmap\\map\\map.go:333", myTIDCache)
		tracer.ReadAcc(&ok, "testcases\\gocHashmap\\map\\map.go:334", myTIDCache)
		if !ok {
			continue
		}
		tracer.ReadAcc(&k, "testcases\\gocHashmap\\map\\map.go:337", myTIDCache)
		tracer.ReadAcc(&v, "testcases\\gocHashmap\\map\\map.go:337", myTIDCache)
		if !f(k, v) {
			break
		}
	}
}

func (m *Map) missLocked() {
	myTIDCache := tracer.GetGID()
	tracer.WriteAcc(&m.misses, "testcases\\gocHashmap\\map\\map.go:344", myTIDCache)
	m.misses++
	tracer.ReadAcc(&m.misses, "testcases\\gocHashmap\\map\\map.go:345", myTIDCache)
	tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:345", myTIDCache)
	if m.misses < len(m.dirty) {
		return
	}
	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:348", myTIDCache)
	m.read.Store(readOnly{m: m.dirty})
	tracer.WriteAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:348", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:348", myTIDCache)
	tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:348", myTIDCache)
	m.dirty = nil
	tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:349", myTIDCache)
	m.misses = 0
	tracer.WriteAcc(&m.misses, "testcases\\gocHashmap\\map\\map.go:350", myTIDCache)
}

func (m *Map) dirtyLocked() {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:354", myTIDCache)
	if m.dirty != nil {
		return
	}

	tracer.PreLock(&m.read, "testcases\\gocHashmap\\map\\map.go:358", myTIDCache)
	read, _ := m.read.Load().(readOnly)
	tracer.ReadAcc(&m.read, "testcases\\gocHashmap\\map\\map.go:358", myTIDCache)
	tracer.PostLock(&m.read, "testcases\\gocHashmap\\map\\map.go:358", myTIDCache)
	tracer.WriteAcc(&read, "testcases\\gocHashmap\\map\\map.go:358", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:359", myTIDCache)
	m.dirty = make(map[interface{}]*entry, len(read.m))
	tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:359", myTIDCache)
	tracer.ReadAcc(&read.m, "testcases\\gocHashmap\\map\\map.go:360", myTIDCache)
	for k, e := range read.m {
		tracer.ReadAcc(&k, "testcases\\gocHashmap\\map\\map.go:361", myTIDCache)
		tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:361", myTIDCache)
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
			tracer.ReadAcc(&e, "testcases\\gocHashmap\\map\\map.go:362:1", myTIDCache)
			tracer.WriteAcc(&m.dirty, "testcases\\gocHashmap\\map\\map.go:362:2", myTIDCache)
		}
	}
}

func (e *entry) tryExpungeLocked() (isExpunged bool) {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:368", myTIDCache)
	tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:368", myTIDCache)
	p := atomic.LoadPointer(&e.p)
	tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:368", myTIDCache)
	tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:368", myTIDCache)
	for p == nil {
		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:370", myTIDCache)
		tracer.WriteAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:370", myTIDCache)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:368", myTIDCache)
		if atomic.CompareAndSwapPointer(&e.p, nil, expunged) {
			return true
		}
		tracer.PreLock(&e.p, "testcases\\gocHashmap\\map\\map.go:373", myTIDCache)
		tracer.ReadAcc(&e.p, "testcases\\gocHashmap\\map\\map.go:373", myTIDCache)
		p = atomic.LoadPointer(&e.p)
		tracer.PostLock(&e.p, "testcases\\gocHashmap\\map\\map.go:373", myTIDCache)
		tracer.WriteAcc(&p, "testcases\\gocHashmap\\map\\map.go:373", myTIDCache)
	}
	return p == expunged
}
