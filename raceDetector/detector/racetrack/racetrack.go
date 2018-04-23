package raceTrack

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
	EXCLUSIVE0    = 1 << iota
	EXCLUSIVE1    = 1 << iota
	SHAREDREAD    = 1 << iota
	SHAREDMODIFY1 = 1 << iota
	EXCLUSIVE2    = 1 << iota
	SHAREDMODIFY2 = 1 << iota
)

func (l *ListenerGoStart) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.IsGoStart {
		return
	}

	t1 := m.Threads[p.T1]
	t2 := m.Threads[p.T2]
	t1.VC.Add(t1.ID, 1)
	t2.VC.Sync(t1.VC)
	t2.VC.Add(t2.ID, 1)
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
	t1.VC.Add(t1.ID, 2) // update thread vc for successful storing (pre+post)
	//update bufferslot by storing the event and sync the vc of the thread and the bufferslot
	ch.Buf[ch.Next].Item = t1.Events[0]

	if ev.Ops[0].Mutex&util.LOCK == 0 {
		//only sync if its not a mutex
		t1.VC.Sync(ch.Buf[ch.Next].VC)
	}

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
			//thread owns this lock now move it to the lockset
			t1.MutexSet[ev.Ops[0].Ch] = struct{}{}
		}

		t1.VC.Add(t1.ID, 2) //update thread vc for succ rcv
		if ev.Ops[0].Mutex&util.LOCK == 0 {
			//no sync for locks
			t1.VC.Sync(ch.Buf[ch.Rnext].VC) //sync thread vc and buffer slot
		}

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
	//	t2.VC.Sync(t1.VC)

	t1.Pop() //pre
	t2.Pop() //pre
	t1.Pop() //post
	t2.Pop() //post

	m.Threads[t1.ID] = t1
	m.Threads[t2.ID] = t2
}
func (l *ListenerDataAccess) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DataAccess {
		return
	}

	thread := m.Threads[p.T1]
	ev := thread.Peek()
	isRead := ev.Ops[0].Kind&util.READ > 0
	varstate := m.Vars3[ev.Ops[0].Ch]
	thread.VC.Add(thread.ID, 1)

	if varstate == nil {
		//handle virgin state here
		varstate = &util.VarState3{Rvc: util.NewVC(), LastAccess: thread.ID,
			LastOp: &ev.Ops[0], State: EXCLUSIVE0}
		m.Vars3[ev.Ops[0].Ch] = varstate
		thread.Pop()
		m.Threads[p.T1] = thread
		return
	}

	switch varstate.State {
	case EXCLUSIVE0:
		if varstate.LastAccess != thread.ID {
			varstate.State = EXCLUSIVE1
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		}
	case EXCLUSIVE1:
		//threadset merge according to RT
		varstate.Rvc = varstate.Rvc.Remove(thread.VC)
		varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		if len(varstate.Rvc) > 1 {
			if isRead {
				varstate.State = SHAREDREAD
			} else {
				varstate.State = SHAREDMODIFY1
			}
		}
	case EXCLUSIVE2:
		//threadset merge according to RT
		varstate.Rvc = varstate.Rvc.Remove(thread.VC)
		varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		if len(varstate.Rvc) > 1 {
			varstate.State = SHAREDMODIFY2
		}
	case SHAREDREAD:
		varstate.MutexSet = eraser.Intersection(varstate.MutexSet, thread.MutexSet)
		if !isRead {
			if len(varstate.MutexSet) > 0 {
				varstate.State = SHAREDMODIFY1
			} else {
				varstate.State = EXCLUSIVE2
				varstate.Rvc = util.NewVC()
				varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
				varstate.MutexSet = thread.MutexSet
			}
		}
	case SHAREDMODIFY1:
		varstate.MutexSet = eraser.Intersection(varstate.MutexSet, thread.MutexSet)
		if len(varstate.MutexSet) == 0 {
			varstate.State = EXCLUSIVE2
			varstate.Rvc = util.NewVC()
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		}
	case SHAREDMODIFY2:
		//threadset merge according to RT
		varstate.Rvc = varstate.Rvc.Remove(thread.VC)
		varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		varstate.MutexSet = eraser.Intersection(varstate.MutexSet, thread.MutexSet)
		if len(varstate.Rvc) == 1 {
			varstate.State = EXCLUSIVE2
		} else if len(varstate.Rvc) > 0 && len(varstate.MutexSet) == 0 {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}
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
