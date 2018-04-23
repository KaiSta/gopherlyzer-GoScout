package eraser

import (
	"../../util"
	"../report"
	"../traceReplay"
)

type ListenerAsyncSnd struct{}
type ListenerAsyncRcv struct{}
type ListenerChanClose struct{}
type ListenerOpClosedChan struct{}
type ListenerSync struct{}
type ListenerDataAccess struct{}
type ListenerSelect struct{}
type ListenerGoStart struct{}

func (l *ListenerGoStart) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.IsGoStart {
		return
	}

	t1 := m.Threads[p.T1]
	t2 := m.Threads[p.T2]
	t1.Pop()
	t2.Pop()
	m.Threads[p.T1] = t1
	m.Threads[p.T2] = t2
}

func (l *ListenerAsyncSnd) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.AsyncSend {
		return
	}

	t1 := m.Threads[p.T1]

	//check if its a mutex.lock or not
	ev := t1.Peek()
	if ev.Ops[0].Mutex&util.LOCK == 0 {
		t1.Pop()
		t1.Pop()
		m.Threads[t1.ID] = t1
		return
	}

	ch := m.AsyncChans2[p.T2]
	if ch.Count == ch.BufSize { //check if there is a free slot
		return
	}

	//update bufferslot by storing the event and sync the vc of the thread and the bufferslot
	ch.Buf[ch.Next].Item = t1.Events[0]

	//thread owns this lock now move it to the lockset
	if t1.MutexSet == nil {
		t1.MutexSet = make(map[string]struct{})
	}
	t1.MutexSet[ev.Ops[0].Ch] = struct{}{}

	//TODO updating the state of the async chan should be handled by the chan itself not here
	ch.Next++ //next free slot
	if ch.Next >= ch.BufSize {
		ch.Next = 0
	}
	ch.Count++

	//remove the top event from the thread stack
	t1.Pop() //pre
	t1.Pop() //post

	//update the traceReplay.Machine state
	m.Threads[t1.ID] = t1
	m.AsyncChans2[p.T2] = ch
}
func (l *ListenerAsyncRcv) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.AsyncRcv {
		return
	}

	//check if its a mutex.unlock or not
	t1 := m.Threads[p.T1]
	ev := t1.Peek()
	if ev.Ops[0].Mutex&util.UNLOCK == 0 {
		t1.Pop()
		t1.Pop()
		m.Threads[t1.ID] = t1
		return
	}

	ch := m.AsyncChans2[p.T2]
	if ch.Count > 0 { //is there something to receive?
		//empty the buffer slot and update the chan state
		ch.Buf[ch.Rnext].Item = nil
		ch.Rnext++ //next slot from which will be received
		if ch.Rnext >= ch.BufSize {
			ch.Rnext = 0
		}
		ch.Count--

		//remove lock from lockset
		if t1.MutexSet == nil {
			t1.MutexSet = make(map[string]struct{})
		}
		if ev.Ops[0].Mutex&util.UNLOCK > 0 {
			//thread releases this lock remove it from lockset
			delete(t1.MutexSet, ev.Ops[0].Ch)
		}

		t1.Pop() //pre
		t1.Pop() //post

		//update the traceReplay.Machine state
		m.Threads[t1.ID] = t1
		m.AsyncChans2[p.T2] = ch
	}
}

func (l *ListenerChanClose) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DoClose {
		return
	}
	t1 := m.Threads[p.T1]
	t1.Pop() //pre
	t1.Pop() //post

	//update the traceReplay.Machine state
	m.Threads[t1.ID] = t1
}
func (l *ListenerOpClosedChan) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.Closed {
		return
	}
	t1 := m.Threads[p.T1]
	t1.Pop() //pre
	t1.Pop() //post

	//update the traceReplay.Machine state
	m.Threads[t1.ID] = t1
}
func (l *ListenerSync) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.Sync {
		return
	}
	t1 := m.Threads[p.T1]
	t2 := m.Threads[p.T2]
	t1.Pop()
	t2.Pop() //pre
	t1.Pop()
	t2.Pop() //post

	//update the traceReplay.Machine state
	m.Threads[t1.ID] = t1
	m.Threads[t2.ID] = t2
}

func (l *ListenerDataAccess) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DataAccess {
		return
	}

	thread := m.Threads[p.T1]
	ev := thread.Peek()

	//get the lockset for this variable
	varstate := m.Vars3[ev.Ops[0].Ch]
	if varstate == nil {
		//variable was touched the first time, set its lockset  to the lockset of
		//the accessing thread to initialize it
		varstate = &util.VarState3{LastAccess: thread.ID, State: util.EXCLUSIVE,
			MutexSet: make(map[string]struct{}), LastOp: &ev.Ops[0]}
		//deepcopy
		for k, v := range thread.MutexSet {
			varstate.MutexSet[k] = v
		}
	}

	isRead := ev.Ops[0].Kind&util.READ > 0

	switch varstate.State {
	case util.EXCLUSIVE:
		if thread.ID != varstate.LastAccess {
			varstate.MutexSet = Intersection(varstate.MutexSet, thread.MutexSet)
			if isRead {
				varstate.State = util.READSHARED
			} else {
				varstate.State = util.SHARED
			}
		}
	case util.READSHARED:
		varstate.MutexSet = Intersection(varstate.MutexSet, thread.MutexSet)
		if !isRead {
			varstate.State = util.SHARED
		}
	case util.SHARED:
		varstate.MutexSet = Intersection(varstate.MutexSet, thread.MutexSet)
	}

	if varstate.State == util.SHARED && len(varstate.MutexSet) == 0 {
		report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
	}

	varstate.LastAccess = thread.ID
	varstate.LastOp = &ev.Ops[0]

	m.Vars3[ev.Ops[0].Ch] = varstate
	thread.Pop()
	m.Threads[p.T1] = thread
}

func (l *ListenerSelect) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.IsSelect {
		return
	}
	// t1 := m.Threads[p.T1]
	// t1.Pop() //pre
	// t1.Pop() //post

	// //update the traceReplay.Machine state
	// m.Threads[t1.ID] = t1
}

func Intersection(m1, m2 map[string]struct{}) map[string]struct{} {
	m3 := make(map[string]struct{})

	for k1 := range m1 {
		_, ok := m2[k1]
		if ok {
			m3[k1] = struct{}{}
		}
	}
	return m3
}
