package twinTrack

import (
	"fmt"

	"../../util"
	"../eraser"
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

const (
	EXCLUSIVE    = 1 << iota
	SHAREDREAD   = 1 << iota
	SHAREDMODIFY = 1 << iota
)

func (l *ListenerAsyncSnd) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.AsyncSend {
		return
	}

	t1 := m.Threads[p.T1]

	ch := m.AsyncChans2[p.T2]
	if ch.Count == ch.BufSize { //check if there is a free slot
		return
	}

	ev := t1.Peek()
	//check if its a mutex operation or a channel op
	if ev.Ops[0].Mutex&util.LOCK > 0 {
		//thread owns this lock now move it to the lockset
		if t1.MutexSet == nil {
			t1.MutexSet = make(map[string]struct{})
		}
		t1.MutexSet[ev.Ops[0].Ch] = struct{}{}
	}
	t1.VC.Add(t1.ID, 2)  // update thread vc for successful storing (pre+post)
	t1.RVC.Add(t1.ID, 2) //racetrack vc
	//update bufferslot by storing the event and sync the vc of the thread and the bufferslot
	ch.Buf[ch.Next].Item = t1.Events[0]
	t1.VC.Sync(ch.Buf[ch.Next].VC)

	// if ev.Ops[0].Mutex&util.LOCK == 0 {
	// 	// no mutex lock, sync rvc
	// 	t1.RVC.Sync(ch.Buf[ch.Next].VC) //really vc and not a second rvc for chans?
	// }

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

	ch := m.AsyncChans2[p.T2]
	if ch.Count > 0 { //is there something to receive?
		t1 := m.Threads[p.T1]
		ev := t1.Peek()
		//check if its a mutex operation or a channel op
		if ev.Ops[0].Mutex&util.UNLOCK > 0 {
			//thread releases this lock remove it from lockset
			delete(t1.MutexSet, ev.Ops[0].Ch)
		}

		t1.VC.Add(t1.ID, 2) //update thread vc for succ rcv
		t1.RVC.Add(t1.ID, 2)
		t1.VC.Sync(ch.Buf[ch.Rnext].VC) //sync thread vc and buffer slot

		// if ev.Ops[0].Mutex&util.UNLOCK == 0 {
		// 	// no mutex lock, sync rvc
		// 	t1.RVC.Sync(ch.Buf[ch.Rnext].VC) //really vc and not a second rvc for chans?
		// }

		//empty the buffer slot and update the chan state
		ch.Buf[ch.Rnext].Item = nil
		ch.Rnext++ //next slot from which will be received
		if ch.Rnext >= ch.BufSize {
			ch.Rnext = 0
		}
		ch.Count--

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

	t1.VC.Add(t1.ID, 2)
	t1.RVC.Add(t1.ID, 2)
	m.ClosedChansVC[p.T2] = t1.VC.Clone()
	m.ClosedChans[p.T2] = struct{}{}
	t1.Pop()
	t1.Pop()
	m.Threads[t1.ID] = t1
}
func (l *ListenerOpClosedChan) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.Closed {
		return
	}
	t1 := m.Threads[p.T1]

	if _, ok := m.ClosedChans[p.T2]; ok && t1.Events[0].Ops[0].Ch == p.T2 &&
		t1.Events[0].Ops[0].Kind&util.SEND > 0 {
		fmt.Println("Send on closed channel!")
	}

	t1.VC.Add(t1.ID, 2)
	t1.RVC.Add(t1.ID, 2)
	t1.Pop()
	t1.Pop()
	m.Threads[t1.ID] = t1
}
func (l *ListenerSync) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.Sync {
		return
	}

	t1 := m.Threads[p.T1]
	t2 := m.Threads[p.T2]

	var ch string
	if len(t1.Events[0].Ops) == 1 {
		ch = t1.Events[0].Ops[0].Ch
	} else {
		ch = t2.Events[0].Ops[0].Ch
	}

	if _, ok := m.ClosedChans[ch]; ok {
		fmt.Println("Send on closed channel!")
	}

	//prepare + commit
	t1.VC.Add(t1.ID, 2)
	t2.VC.Add(t2.ID, 2)
	t1.VC.Sync(t2.VC) //sync updates both

	//rvc sync for channel ops
	t1.RVC.Add(t1.ID, 2)
	t2.RVC.Add(t2.ID, 2)
	t1.RVC.Sync(t2.RVC)
	//	t2.VC.Sync(t1.VC)

	t1.Pop() //pre
	t2.Pop() //pre
	t1.Pop() //post
	t2.Pop() //post

	m.Threads[t1.ID] = t1
	m.Threads[t2.ID] = t2
}

func fastTrackMethod(m *traceReplay.Machine, p *traceReplay.SyncPair) bool {
	foundRace := false
	thread := m.Threads[p.T1]
	ev := thread.Peek()
	thread.VC.Add(thread.ID, 1)
	varstate := m.Vars3[ev.Ops[0].Ch]

	if varstate == nil {
		varstate = &util.VarState3{Rvc: util.NewVC(), Wvc: util.NewVC(),
			LastAccess: thread.ID, State: util.EXCLUSIVE,
			LastOp: &ev.Ops[0]}
	}

	if ev.Ops[0].Kind&util.WRITE > 0 {
		if !varstate.Wvc.Less(thread.VC) {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
			foundRace = true
			varstate.Wvc.Set(thread.ID, thread.VC[thread.ID])
		} else {
			varstate.Wvc = util.NewVC()
			varstate.Wvc.Set(thread.ID, thread.VC[thread.ID])
		}
		if !varstate.Rvc.Less(thread.VC) {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
			foundRace = true
		}
	} else if ev.Ops[0].Kind&util.READ > 0 {
		if !varstate.Wvc.Less(thread.VC) {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
			foundRace = true
		}
		if !varstate.Rvc.Less(thread.VC) {
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		} else {
			varstate.Rvc = util.NewVC()
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		}
	}

	//varstate.LastAccess = thread.ID
	//	varstate.LastOp = &ev.Ops[0]

	m.Vars3[ev.Ops[0].Ch] = varstate
	//thread.Pop()  leave out pop operation so the listener can handle the event agian for the rest of the algorithm
	m.Threads[p.T1] = thread

	return foundRace
}

func (l *ListenerDataAccess) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DataAccess {
		return
	}
	thread := m.Threads[p.T1]
	if fastTrackMethod(m, p) {
		thread.Pop()
		m.Threads[p.T1] = thread
		return
	}
	thread.RVC.Add(thread.ID, 1)
	ev := thread.Peek()
	isRead := ev.Ops[0].Kind&util.READ > 0
	varstate := m.Vars3[ev.Ops[0].Ch]

	//separate reads don't change the mutexset, if
	// the first access is a read it does not initialize
	// the mutexset so the first write can do it instead
	// while assuming that the var is protected by all locks
	if !isRead && varstate.MutexSet == nil {
		varstate.MutexSet = make(map[string]struct{})
		for k, v := range thread.MutexSet {
			varstate.MutexSet[k] = v
		}
	}

	if varstate.TSet == nil {
		//handle virgin state here
		varstate.TSet = util.NewVC()
		varstate.TSet.Set(thread.ID, thread.VC[thread.ID])
		m.Vars3[ev.Ops[0].Ch] = varstate
		thread.Pop()
		m.Threads[p.T1] = thread
		return
	}

	varstate.TSet = varstate.TSet.Remove(thread.RVC)
	varstate.TSet.Set(thread.ID, thread.VC[thread.ID])

	//only writes update the mutexset
	tmp := eraser.Intersection(varstate.MutexSet, thread.MutexSet)
	//varstate.MutexSet = eraser.Intersection(varstate.MutexSet, thread.MutexSet)

	switch varstate.State {
	case EXCLUSIVE:
		if len(varstate.TSet) > 1 {
			if isRead {
				varstate.State = SHAREDREAD
			} else {
				varstate.State = SHAREDMODIFY
			}
		}

	case SHAREDREAD:
		if !isRead {
			varstate.State = SHAREDMODIFY
		}
	}

	if varstate.State == SHAREDMODIFY && len(tmp) == 0 &&
		len(varstate.TSet) > 1 {
		report.Race(varstate.LastOp, &ev.Ops[0], report.LOW)
	}

	if !isRead {
		varstate.MutexSet = tmp
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

	threadSelect := m.Threads[p.T1]
	//	if p.closed { //default case

	vc := threadSelect.VC.Clone()
	vc.Add(p.T1, 1)
	it := threadSelect.Events[0]
	m.Selects = append(m.Selects, traceReplay.SelectStore{vc, it})
}

func (l *ListenerGoStart) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.IsGoStart {
		return
	}

	t1 := m.Threads[p.T1]
	t2 := m.Threads[p.T2]
	t1.VC.Add(t1.ID, 1)
	t1.RVC.Add(t1.ID, 1)
	t2.VC.Sync(t1.VC)
	t2.VC.Add(t2.ID, 1)
	t2.RVC.Sync(t1.RVC)
	t2.RVC.Add(t2.ID, 1)
	t1.Pop()
	t2.Pop()
	m.Threads[p.T1] = t1
	m.Threads[p.T2] = t2
}
