package fastTrack

import (
	"fmt"

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
type ListenerDataAccess2 struct{}
type ListenerSelect struct{}
type ListenerGoStart struct{}

func handleLock(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	t1 := m.Threads[p.T1]
	ch := m.AsyncChans2[p.T2]
	ev := t1.Peek()

	pre := ev.Clone()
	pre.VC = t1.VC.Clone()
	pre.VC.Add(t1.ID, 1)

	if ev.Ops[0].Mutex&util.LOCK > 0 {
		if ch.Count > 0 {
			return
		}
		t1.VC.Add(t1.ID, 2)
		ch.Buf[0].Item = ev
		ch.Buf[1].Item = ev
		t1.VC.Sync(ch.Buf[0].VC)
		t1.VC.Sync(ch.Buf[1].VC)
		ch.Count = 2
	} else if ev.Ops[0].Mutex&util.UNLOCK > 0 {
		if ch.Count != 2 {
			return
		}
		t1.VC.Add(t1.ID, 2)
		t1.VC.Sync(ch.Buf[0].VC)
		t1.VC.Sync(ch.Buf[1].VC)
		ch.Buf[0].Item = nil
		ch.Buf[1].Item = nil
		ch.Count = 0
	} else if ev.Ops[0].Mutex&util.RLOCK > 0 {
		if ch.Count > 1 {
			return //writer lock already claimed both fields
		}
		t1.VC.Add(t1.ID, 2)
		t1.VC.Sync(ch.Buf[1].VC) //sync with writer vc
		ch.Buf[0].Item = nil
		ch.Rcounter++
		if ch.Count == 0 { //first reader
			ch.Count = 1
		}
	} else if ev.Ops[0].Mutex&util.RUNLOCK > 0 {
		if ch.Count > 1 {
			return // lock was a writer lock not a readlock
		}
		t1.VC.Add(t1.ID, 2)
		t1.VC.Sync(ch.Buf[0].VC) // sync with reader vc
		ch.Buf[0].Item = nil
		ch.Rcounter--
		if ch.Rcounter == 0 { //last reader
			ch.Count = 0
		}
	}

	t1.Pop()
	post := t1.Peek().Clone()
	post.VC = t1.VC.Clone()
	report.AddEvent(pre, post)

	t1.Pop()
	m.Threads[p.T1] = t1
	m.AsyncChans2[p.T2] = ch
}

func (l *ListenerAsyncSnd) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.AsyncSend {
		return
	}

	t1 := m.Threads[p.T1]

	ev := t1.Peek()
	if ev.Ops[0].Mutex&util.LOCK > 0 || ev.Ops[0].Mutex&util.RLOCK > 0 {
		handleLock(m, p)
		return
	}

	pre := ev.Clone()
	pre.VC = t1.VC.Clone()
	pre.VC.Add(t1.ID, 1)

	ch := m.AsyncChans2[p.T2]
	if ch.Count == ch.BufSize { //check if there is a free slot
		return
	}
	if _, ok := m.ClosedChans[p.T2]; ok {
		fmt.Println("Send on closed channel!")
	}

	t1.VC.Add(t1.ID, 2) // update thread vc for successful storing (pre+post)
	//update bufferslot by storing the event and sync the vc of the thread and the bufferslot
	ch.Buf[ch.Next].Item = t1.Events[0]
	t1.VC.Sync(ch.Buf[ch.Next].VC)

	//TODO updating the state of the async chan should be handled by the chan itself not here
	ch.Next++ //next free slot
	if ch.Next >= ch.BufSize {
		ch.Next = 0
	}
	ch.Count++

	// cvc := m.ChanVC[p.t2]
	// cvc.Wvc.Sync(t1.VC.Clone())
	// m.ChanVC[p.t2] = cvc

	//remove the top event from the thread stack
	// ev.VC = t1.VC.Clone()
	// report.AddEvent(ev)

	t1.Pop() //pre
	post := t1.Peek().Clone()
	post.VC = t1.VC.Clone()
	report.AddEvent(pre, post)
	t1.Pop() //post

	//update the traceReplay.Machine state
	m.Threads[t1.ID] = t1
	m.AsyncChans2[p.T2] = ch
}
func (l *ListenerAsyncRcv) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.AsyncRcv {
		return
	}
	t1 := m.Threads[p.T1]

	ev := t1.Peek()
	if ev.Ops[0].Mutex&util.UNLOCK > 0 || ev.Ops[0].Mutex&util.RUNLOCK > 0 {
		handleLock(m, p)
		return
	}

	pre := ev.Clone()
	pre.VC = t1.VC.Clone()
	pre.VC.Add(t1.ID, 1)

	ch := m.AsyncChans2[p.T2]
	if ch.Count > 0 { //is there something to receive?
		t1 := m.Threads[p.T1]
		t1.VC.Add(t1.ID, 2)             //update thread vc for succ rcv
		t1.VC.Sync(ch.Buf[ch.Rnext].VC) //sync thread vc and buffer slot

		//empty the buffer slot and update the chan state
		ch.Buf[ch.Rnext].Item = nil
		ch.Rnext++ //next slot from which will be received
		if ch.Rnext >= ch.BufSize {
			ch.Rnext = 0
		}
		ch.Count--

		// ev.VC = t1.VC.Clone()
		// report.AddEvent(ev)
		t1.Pop() //pre
		post := t1.Peek().Clone()
		post.VC = t1.VC.Clone()
		report.AddEvent(pre, post)
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

	pre := t1.Peek().Clone()
	pre.VC = t1.VC.Clone()
	pre.VC.Add(t1.ID, 1)
	report.AddEvent(pre, nil) //no partner

	cvc := m.ChanVC[p.T2]
	if cvc != nil && !cvc.Wvc.Less(t1.VC) {
		fmt.Println("Concurrent send during close operation on channel ", p.T2)
	}

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

	if p.T2 == "0" { // select default case
		nit := &util.Item{Thread: "0", Partner: t1.ID, VC: t1.VC.Clone()}
		nit.VC.Add(t1.ID, 1)
		nit.Ops = append(nit.Ops, util.Operation{Ch: "0", Kind: util.PREPARE | util.SEND})
		report.AddEvent(nit, nil)
	} else {
		tmp := t1.Peek().Clone()
		tmp.VC = t1.VC.Clone()
		tmp.VC.Add(t1.ID, 1)
		report.AddEvent(tmp, nil)
	}

	if _, ok := m.ClosedChans[p.T2]; ok && t1.Events[0].Ops[0].Ch == p.T2 &&
		t1.Events[0].Ops[0].Kind&util.SEND > 0 {
		fmt.Println("Send on closed channel!")
	}

	t1.VC.Add(t1.ID, 2)
	// ev := t1.Peek()
	// ev.VC = t1.VC.Clone()
	// report.AddEvent(ev)
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

	pre := t1.Peek().Clone()
	pre2 := t2.Peek().Clone()
	pre.VC = t1.VC.Clone()
	pre2.VC = t2.VC.Clone()
	pre.VC.Add(t1.ID, 1)
	pre2.VC.Add(t2.ID, 1)

	var ch string
	if len(t1.Events[0].Ops) == 1 {
		ch = t1.Events[0].Ops[0].Ch
	} else {
		ch = t2.Events[0].Ops[0].Ch
	}

	if _, ok := m.ClosedChans[ch]; ok {
		fmt.Println("Send on closed channel!")
	}
	for _, s := range m.Selects {
		for _, x := range s.Ev.Ops {
			dual := util.OpKind(util.PREPARE | util.SEND)
			if x.Kind == util.PREPARE|util.SEND {
				dual = util.PREPARE | util.RCV
			}
			if x.Ch == ch && dual == t1.Events[0].Ops[0].Kind {
				tmp := fmt.Sprintf("%v", s.Ev)
				report.SelectAlternative(tmp, t1.Events[0].String())
				//	fmt.Printf("2Event %v is an alternative for select statement \n\t%v\n", t1.Events[0], s.Ev)
			} else if x.Ch == ch && dual == t2.Events[0].Ops[0].Kind {
				tmp := fmt.Sprintf("%v", s.Ev)
				tmp2 := fmt.Sprintf("%v", t2.Events[0])
				report.SelectAlternative(tmp, tmp2)
				//	fmt.Printf("3Event %v is an alternative for select statement \n\t%v\n", t2.Events[0], s.Ev)
			}
		}
	}

	//prepare + commit
	t1.VC.Add(t1.ID, 2)
	t2.VC.Add(t2.ID, 2)
	t1.VC.Sync(t2.VC) //sync updates both
	//	t2.VC.Sync(t1.VC)
	t1.Pop() //pre
	t2.Pop() //pre

	post := t1.Peek().Clone()
	post.VC = t1.VC.Clone()
	post2 := t2.Peek().Clone()
	post2.VC = t2.VC.Clone()
	report.AddEvent(pre, post)
	report.AddEvent(pre2, post)

	t1.Pop() //post
	t2.Pop() //post

	m.Threads[t1.ID] = t1
	m.Threads[t2.ID] = t2
}

func (l *ListenerDataAccess2) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DataAccess {
		return
	}
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
			varstate.Wvc.Set(thread.ID, thread.VC[thread.ID])
		} else {
			varstate.Wvc = util.NewVC()
			varstate.Wvc.Set(thread.ID, thread.VC[thread.ID])
		}
		if !varstate.Rvc.Less(thread.VC) {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}
	} else if ev.Ops[0].Kind&util.READ > 0 {
		if !varstate.Wvc.Less(thread.VC) {
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}
		if !varstate.Rvc.Less(thread.VC) {
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		} else {
			varstate.Rvc = util.NewVC()
			varstate.Rvc.Set(thread.ID, thread.VC[thread.ID])
		}
	}

	varstate.LastAccess = thread.ID
	varstate.LastOp = &ev.Ops[0]

	m.Vars3[ev.Ops[0].Ch] = varstate
	thread.Pop()
	m.Threads[p.T1] = thread
}

func (l *ListenerDataAccess) Put(m *traceReplay.Machine, p *traceReplay.SyncPair) {
	if !p.DataAccess {
		return
	}

	thread := m.Threads[p.T1]
	ev := thread.Peek()
	if ev.Ops[0].Kind&util.WRITE > 0 {
		thread.VC.Add(thread.ID, 1)

		varstate := m.Vars3[ev.Ops[0].Ch]

		if varstate == nil {
			varstate = &util.VarState3{Rvc: nil, Wvc: nil, Wepoch: util.NewEpoch(thread.ID, 0), Repoch: util.NewEpoch(thread.ID, -1), LastAccess: thread.ID, State: util.EXCLUSIVE,
				LastOp: &ev.Ops[0]}
		}

		if varstate.Wvc == nil && !varstate.Wepoch.Less_Epoch(thread.VC) {
			//fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.PrevWEv, varstate.PrevWEv)
			//fmt.Println(1, varstate.Wepoch, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
			varstate.Wvc = util.NewVC()
			varstate.Wvc[thread.ID] = thread.VC[thread.ID]
			varstate.Wvc[varstate.Wepoch.X] = varstate.Wepoch.T
		} else if varstate.Wvc != nil && !varstate.Wvc.Less(thread.VC) {
			//fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.PrevWEv, varstate.Wvc.FindConflict(thread.VC))
			//fmt.Println(2, varstate.Wvc, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}
		if varstate.Wvc != nil && varstate.Wvc.Less(thread.VC) {
			varstate.Wvc = nil
		}

		if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(thread.VC) {
			//fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.PrevREv, varstate.PrevREv)
			//fmt.Println(3, varstate.Repoch, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		} else if varstate.Rvc != nil && !varstate.Rvc.Less(thread.VC) {
			//fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.Wvc.FindConflict(thread.VC), varstate.PrevREv)
			//fmt.Println(4, varstate.Rvc, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}

		if varstate.Wvc == nil {
			varstate.Wepoch.Set(thread.ID, thread.VC[thread.ID])
		} else {
			varstate.Wvc.Sync(thread.VC.Clone())
		}

		varstate.LastAccess = thread.ID
		varstate.LastOp = &ev.Ops[0]

		varstate.PrevWEv = ev
		m.Vars3[ev.Ops[0].Ch] = varstate
		thread.Pop()
		m.Threads[p.T1] = thread
	} else if ev.Ops[0].Kind&util.READ > 0 {
		thread.VC.Add(thread.ID, 1)

		varstate := m.Vars3[ev.Ops[0].Ch]

		if varstate == nil {
			varstate = &util.VarState3{Rvc: nil, Wepoch: util.NewEpoch(thread.ID, -1), Repoch: util.NewEpoch(thread.ID, 0), LastAccess: thread.ID, State: util.EXCLUSIVE,
				LastOp: &ev.Ops[0]}
		}

		if varstate.Wvc == nil && !varstate.Wepoch.Less_Epoch(thread.VC) {
			//fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.PrevREv)
			//fmt.Println(5, varstate.Wepoch, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		} else if varstate.Wvc != nil && !varstate.Wvc.Less(thread.VC) {
			//	fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nlast Access: %v\n\n", ev.Ops[0].Ch, thread.ID, ev, varstate.PrevREv)
			//fmt.Println(6, varstate.Wvc, "||", thread.VC)
			report.Race(varstate.LastOp, &ev.Ops[0], report.SEVERE)
		}

		if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(thread.VC) {
			// current read is concurrent to the epoch stored for the var. create a new VC and use Rvc for a while
			varstate.Rvc = util.NewVC()
			varstate.Rvc[thread.ID] = thread.VC[thread.ID]
			varstate.Rvc[varstate.Repoch.X] = varstate.Repoch.T
		}

		if varstate.Rvc != nil && varstate.Rvc.Less(thread.VC) {
			// used read vector clock so far, the current read comes after the previously concurrent reads,
			// switch back to read epoch
			varstate.Rvc = nil
		}

		if varstate.Rvc != nil {
			varstate.Rvc.Sync(thread.VC.Clone())
		} else {
			varstate.Repoch.Set(thread.ID, thread.VC[thread.ID])
		}
		varstate.PrevREv = ev

		// if varstate.State == util.EXCLUSIVE {
		// 	if varstate.LastAccess != thread.ID {
		// 		varstate.State = util.READSHARED
		// 	}
		// }
		varstate.LastAccess = thread.ID
		varstate.LastOp = &ev.Ops[0]

		m.Vars3[ev.Ops[0].Ch] = varstate
		thread.Pop()
		m.Threads[p.T1] = thread
	}
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
	ev := threadSelect.Peek().Clone()
	ev.VC = threadSelect.VC.Clone()
	ev.VC.Add(threadSelect.ID, 1)
	report.AddEvent(ev, nil)
	//} else { //chan op
	//	fmt.Println("2>>>", p)
	//	fmt.Println("SELECT")
	//}
}

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
