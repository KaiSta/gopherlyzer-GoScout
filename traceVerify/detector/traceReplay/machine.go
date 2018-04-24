package traceReplay

import (
	"fmt"

	"../../util"
	"../report"
)

type EventListener interface {
	Put(*Machine, *SyncPair)
}

var EvListener = []EventListener{}

type Machine struct {
	Threads       map[string]util.Thread
	AsyncChans    map[string][]*util.Item
	ClosedChans   map[string]struct{}
	ClosedChansVC map[string]util.VectorClock
	Stopped       bool
	Vars1         map[string]*util.VarState1
	Vars2         map[string]*util.VarState2
	Vars3         map[string]*util.VarState3
	AsyncChans2   map[string]AsyncChan
	ChanVC        map[string]*util.ChanState
	Selects       []SelectStore
}

type SelectStore struct {
	VC util.VectorClock
	Ev *util.Item
}

type AsyncChan struct {
	Buf      []BufField
	BufSize  int
	Next     int
	Count    int
	Rnext    int
	IsRWLock bool
	Rcounter int
}

type BufField struct {
	*util.Item
	VC util.VectorClock
}

func (m Machine) Clone() Machine {
	threads := make(map[string]util.Thread)
	for k, v := range m.Threads {
		threads[k] = v.Clone()
	}
	asyncChans := make(map[string][]*util.Item)
	for k, v := range m.AsyncChans {
		var ops []*util.Item
		for _, i := range v {
			ops = append(ops, i.Clone())
		}
		asyncChans[k] = ops
	}
	closedChans := make(map[string]struct{})
	for k, v := range m.ClosedChans {
		closedChans[k] = v
	}
	return Machine{Threads: threads, AsyncChans: asyncChans, ClosedChans: closedChans, ClosedChansVC: nil, Stopped: false}
}

func (m *Machine) GetSyncPairs() (ret []SyncPair) {
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek()
		for i, op := range e.Ops {
			if op.Kind&util.CLS > 0 {
				ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, DoClose: true, Idx: i})
				continue
			}
			if op.Kind&util.RCV == 0 || op.BufSize > 0 {
				continue
			}
			if _, ok := m.ClosedChans[op.Ch]; ok {
				//closed channel
				ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, Closed: true, Idx: i})
				continue
			}
			for _, p := range m.Threads {
				if p.ID == t.ID || len(p.Events) == 0 {
					continue
				}
				pe := p.Peek()
				for j, pop := range pe.Ops {
					if pop.Ch == op.Ch && pop.Kind == util.OpKind(util.PREPARE|util.SEND) {
						ret = append(ret, SyncPair{T1: t.ID, T2: p.ID, Idx: i, T2Idx: j})
					}
				}
			}

		}
	}
	return
}

func (m *Machine) GetAsyncActions() (ret []SyncPair) {
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek()

		for _, op := range e.Ops {
			if op.Kind == util.OpKind(util.PREPARE|util.SEND) && op.BufSize > 0 {
				c := m.AsyncChans2[op.Ch]
				if c.Count <= c.BufSize {
					ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, AsyncSend: true})
				}
			} else if op.Kind == util.OpKind(util.PREPARE|util.RCV) && op.BufSize > 0 {
				c := m.AsyncChans2[op.Ch]
				if c.Count > 0 {
					ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true})
				}
			}
		}
	}
	return
}

func (m *Machine) GetThreadStarts() (ret map[string]string) {
	ret = make(map[string]string)
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek()
		for _, op := range e.Ops {
			partner := util.OpKind(util.SIG)
			if op.Kind&util.SIG > 0 {
				partner = util.WAIT
			}
			for _, j := range m.Threads {
				if len(j.Events) == 0 {
					continue
				}
				for _, op2 := range j.Peek().Ops {
					if op2.Kind&partner > 0 && op.Ch == op2.Ch {
						if op.Kind&util.SIG > 0 {
							ret[t.ID] = j.ID
							//ret = append(ret, SyncPair{t1: t.ID, t2: j.ID, GoStart: true})
						} else {
							ret[j.ID] = t.ID
							//	ret = append(ret, SyncPair{t1: j.ID, t2: t.ID, GoStart: true})
						}
					}
				}
			}
		}
	}
	return
}

func (m *Machine) StartAllthreads() {
	for {
		threads := m.GetThreadStarts()
		if len(threads) == 0 {
			return
		}

		for k, v := range threads {
			t1 := m.Threads[k]
			t2 := m.Threads[v]
			t1.VC.Add(t1.ID, 1)
			t2.VC.Sync(t1.VC)
			t2.VC.Add(t2.ID, 1) //?
			t1.Pop()
			t2.Pop()
			m.Threads[k] = t1
			m.Threads[v] = t2
		}
	}
}

func (m *Machine) ClosedRcv(p SyncPair, rcvOnClosed map[string]struct{}) {
	if p.Closed {
		t := m.Threads[p.T1]
		if p.T2 != "0" {
			rcvOnClosed[t.Peek().String()] = struct{}{}
		}

		t.Pop()
		t.Pop()
		m.Threads[p.T1] = t
	}
}

func (m *Machine) CloseChan(p SyncPair) {
	t := m.Threads[p.T1]
	t.Pop()
	t.Pop()
	m.Threads[p.T1] = t
	m.ClosedChans[p.T2] = struct{}{}
}

func (m *Machine) ExAllRW() []*util.Item {
	var rw []*util.Item
	for k, v := range m.Threads {
		if len(v.Events) == 0 {
			continue
		}
		for {
			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE || ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)
				v.Events[0].VC = v.VC.Clone()
				rw = append(rw, v.Events[0])
				v.Pop()
				m.Threads[k] = v
			} else {
				break
			}
		}
	}
	return rw
}

func (m *Machine) ExAllRWVC() {
	for k, v := range m.Threads {
		if len(v.Events) == 0 {
			continue
		}
		for {
			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars1[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState1{Rvc: util.NewVC(), Wvc: util.NewVC()}
				}

				if !varstate.Wvc.Less(v.VC) {
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv)
				}
				if !varstate.Rvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				varstate.Wvc = v.VC.Clone()
				varstate.PrevWEv = ev
				m.Vars1[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else if ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars1[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState1{Rvc: util.NewVC(), Wvc: util.NewVC()}
				}

				if !varstate.Wvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv)
				}

				varstate.Rvc.Sync(v.VC.Clone())
				varstate.PrevREv = ev
				m.Vars1[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else {
				break
			}
		}
	}
}
func (m *Machine) ExAllRWEpoch2() {
	for {
		done := 0

		for k, v := range m.Threads {
			if len(v.Events) == 0 {
				continue
			}

			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars2[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState2{Rvc: nil, Wepoch: util.NewEpoch(v.ID, 0), Repoch: util.NewEpoch(v.ID, -1)}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					//fmt.Println(">>", varstate.Wepoch, v.VC)
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv)
				}
				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					//	fmt.Println(">>", varstate.Repoch, v.VC)
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				} else if !varstate.Rvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				varstate.Wepoch.Set(v.ID, v.VC[v.ID])
				varstate.PrevWEv = ev
				m.Vars2[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else if ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars2[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState2{Rvc: nil, Wepoch: util.NewEpoch(v.ID, -1), Repoch: util.NewEpoch(v.ID, 0)}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					// current read is concurrent to the epoch stored for the var. create a new VC and use Rvc for a while
					varstate.Rvc = util.NewVC()
					varstate.Rvc[v.ID] = v.VC[v.ID]
					varstate.Rvc[varstate.Repoch.X] = varstate.Repoch.T
				}

				if varstate.Rvc != nil && varstate.Rvc.Less(v.VC) {
					// used read vector clock so far, the current read comes after the previously concurrent reads,
					// switch back to read epoch
					varstate.Rvc = nil
				}

				if varstate.Rvc != nil {
					varstate.Rvc.Sync(v.VC.Clone())
				} else {
					varstate.Repoch.Set(v.ID, v.VC[v.ID])
				}
				varstate.PrevREv = ev
				m.Vars2[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else {
				done++
			}
		}

		if done == len(m.Threads) {
			return
		}
	}
}

func (m *Machine) ExAllRWEpoch() {
	for k, v := range m.Threads {
		if len(v.Events) == 0 {
			continue
		}
		for {
			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars2[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState2{Rvc: nil, Wepoch: util.NewEpoch(v.ID, 0), Repoch: util.NewEpoch(v.ID, -1)}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					//fmt.Println(">>", varstate.Wepoch, v.VC)
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv)
				}
				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					//	fmt.Println(">>", varstate.Repoch, v.VC)
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				} else if !varstate.Rvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				varstate.Wepoch.Set(v.ID, v.VC[v.ID])
				varstate.PrevWEv = ev
				m.Vars2[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else if ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars2[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState2{Rvc: nil, Wepoch: util.NewEpoch(v.ID, -1), Repoch: util.NewEpoch(v.ID, 0)}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					// current read is concurrent to the epoch stored for the var. create a new VC and use Rvc for a while
					varstate.Rvc = util.NewVC()
					varstate.Rvc[v.ID] = v.VC[v.ID]
					varstate.Rvc[varstate.Repoch.X] = varstate.Repoch.T
				}

				if varstate.Rvc != nil && varstate.Rvc.Less(v.VC) {
					// used read vector clock so far, the current read comes after the previously concurrent reads,
					// switch back to read epoch
					varstate.Rvc = nil
				}

				if varstate.Rvc != nil {
					varstate.Rvc.Sync(v.VC.Clone())
				} else {
					varstate.Repoch.Set(v.ID, v.VC[v.ID])
				}
				varstate.PrevREv = ev
				m.Vars2[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else {
				break
			}
		}
	}
}

func (m *Machine) ExAllRWEpochAndEraser2() {
	for {
		done := 0

		for k, v := range m.Threads {
			if len(v.Events) == 0 {
				done++
				continue
			}

			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars3[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState3{Rvc: nil, Wvc: nil, Wepoch: util.NewEpoch(v.ID, 0), Repoch: util.NewEpoch(v.ID, -1), LastAccess: v.ID, State: util.EXCLUSIVE}
				}

				if varstate.Wvc == nil && !varstate.Wepoch.Less_Epoch(v.VC) {
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv, varstate.PrevWEv)
					varstate.Wvc = util.NewVC()
					varstate.Wvc[v.ID] = v.VC[v.ID]
					varstate.Wvc[varstate.Wepoch.X] = varstate.Wepoch.T
				} else if varstate.Wvc != nil && !varstate.Wvc.Less(v.VC) {
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv, varstate.Wvc.FindConflict(v.VC))

				}
				if varstate.Wvc != nil && varstate.Wvc.Less(v.VC) {
					varstate.Wvc = nil
				}

				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv, varstate.PrevREv)
				} else if varstate.Rvc != nil && !varstate.Rvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nConflict with: %v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.Wvc.FindConflict(v.VC), varstate.PrevREv)
				}

				if varstate.Wvc == nil {
					varstate.Wepoch.Set(v.ID, v.VC[v.ID])
				} else {
					varstate.Wvc.Sync(v.VC.Clone())
				}

				varstate.PrevWEv = ev

				//update Eraser state
				if varstate.State == util.EXCLUSIVE {
					if varstate.LastAccess != v.ID {
						varstate.State = util.SHARED
					}
				} else if varstate.State == util.READSHARED {
					varstate.State = util.SHARED
				}
				varstate.LastAccess = v.ID

				//eraser(varstate, &v, &ev)

				m.Vars3[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v

			} else if ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars3[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState3{Rvc: nil, Wepoch: util.NewEpoch(v.ID, -1), Repoch: util.NewEpoch(v.ID, 0), LastAccess: v.ID, State: util.EXCLUSIVE}
				}

				if varstate.Wvc == nil && !varstate.Wepoch.Less_Epoch(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				} else if varstate.Wvc != nil && !varstate.Wvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nlast Access: %v\n\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					// current read is concurrent to the epoch stored for the var. create a new VC and use Rvc for a while
					varstate.Rvc = util.NewVC()
					varstate.Rvc[v.ID] = v.VC[v.ID]
					varstate.Rvc[varstate.Repoch.X] = varstate.Repoch.T
				}

				if varstate.Rvc != nil && varstate.Rvc.Less(v.VC) {
					// used read vector clock so far, the current read comes after the previously concurrent reads,
					// switch back to read epoch
					varstate.Rvc = nil
				}

				if varstate.Rvc != nil {
					varstate.Rvc.Sync(v.VC.Clone())
				} else {
					varstate.Repoch.Set(v.ID, v.VC[v.ID])
				}
				varstate.PrevREv = ev

				if varstate.State == util.EXCLUSIVE {
					if varstate.LastAccess != v.ID {
						varstate.State = util.READSHARED
					}
				}
				varstate.LastAccess = v.ID

				//eraser(varstate, &v, ev)

				m.Vars3[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else {
				done++
			}
		}

		if done == len(m.Threads) {
			return
		}
	}
}

func (m *Machine) ExAllRWEpochAndEraser() {
	for k, v := range m.Threads {
		if len(v.Events) == 0 {
			continue
		}
		for {
			ev := v.Peek()
			if ev.Ops[0].Kind == util.COMMIT|util.WRITE {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars3[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState3{Rvc: nil, Wepoch: util.NewEpoch(v.ID, 0), Repoch: util.NewEpoch(v.ID, -1), LastAccess: v.ID, State: util.EXCLUSIVE}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					//fmt.Println(">>", varstate.Wepoch, v.VC)
					fmt.Printf("Write-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevWEv)
				}
				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					//	fmt.Println(">>", varstate.Repoch, v.VC)
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				} else if !varstate.Rvc.Less(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				varstate.Wepoch.Set(v.ID, v.VC[v.ID])
				varstate.PrevWEv = ev

				//update Eraser state
				if varstate.State == util.EXCLUSIVE {
					if varstate.LastAccess != v.ID {
						varstate.State = util.SHARED
					}
				} else if varstate.State == util.READSHARED {
					varstate.State = util.SHARED
				}
				varstate.LastAccess = v.ID

				//	eraser(varstate, &v, ev)

				m.Vars3[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v

			} else if ev.Ops[0].Kind == util.COMMIT|util.READ {
				v.VC.Add(v.ID, 1)

				varstate := m.Vars3[ev.Ops[0].Ch]

				if varstate == nil {
					varstate = &util.VarState3{Rvc: nil, Wepoch: util.NewEpoch(v.ID, -1), Repoch: util.NewEpoch(v.ID, 0), LastAccess: v.ID, State: util.EXCLUSIVE}
				}

				if !varstate.Wepoch.Less_Epoch(v.VC) {
					fmt.Printf("Read-Write Race on var %v when thread %v accessed.\n@%v\nwith %v\n", ev.Ops[0].Ch, v.ID, ev, varstate.PrevREv)
				}

				if varstate.Rvc == nil && !varstate.Repoch.Less_Epoch(v.VC) {
					// current read is concurrent to the epoch stored for the var. create a new VC and use Rvc for a while
					varstate.Rvc = util.NewVC()
					varstate.Rvc[v.ID] = v.VC[v.ID]
					varstate.Rvc[varstate.Repoch.X] = varstate.Repoch.T
				}

				if varstate.Rvc != nil && varstate.Rvc.Less(v.VC) {
					// used read vector clock so far, the current read comes after the previously concurrent reads,
					// switch back to read epoch
					varstate.Rvc = nil
				}

				if varstate.Rvc != nil {
					varstate.Rvc.Sync(v.VC.Clone())
				} else {
					varstate.Repoch.Set(v.ID, v.VC[v.ID])
				}
				varstate.PrevREv = ev

				if varstate.State == util.EXCLUSIVE {
					if varstate.LastAccess != v.ID {
						varstate.State = util.READSHARED
					}
				}
				varstate.LastAccess = v.ID

				//	eraser(varstate, &v, ev)

				m.Vars3[ev.Ops[0].Ch] = varstate

				v.Pop()
				m.Threads[k] = v
			} else {
				break
			}
		}
	}
}

//IMPORTANT NEW PARTS
func (m *Machine) GetNextAction() (ret []SyncPair) {
	var async []SyncPair
	var sync []SyncPair
	var dataAccess []SyncPair
	var close []SyncPair
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek()

		for i, op := range e.Ops {
			if op.BufSize > 0 && len(async) == 0 { //async op
				if op.Kind == util.PREPARE|util.SEND {
					c := m.AsyncChans2[op.Ch]
					if c.Count < c.BufSize {
						async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncSend: true})
					}
				} else if op.Kind == util.PREPARE|util.RCV {
					c := m.AsyncChans2[op.Ch]
					if c.Count > 0 {
						async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true})
					}
				}
			} else if (op.Kind&util.WRITE > 0 || op.Kind&util.READ > 0) && len(dataAccess) == 0 { //data access
				dataAccess = append(dataAccess, SyncPair{T1: t.ID, DataAccess: true, Idx: i})
			} else if op.Kind&util.CLS > 0 && len(close) == 0 { //channel closing
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, DoClose: true, Idx: i})
			} else if op.Kind&util.RCV > 0 {
				for _, p := range m.Threads {
					if p.ID == t.ID || len(p.Events) == 0 {
						continue
					}
					pe := p.Peek()
					for j, pop := range pe.Ops {
						if pop.Ch == op.Ch && pop.Kind == util.PREPARE|util.SEND {
							sync = append(sync, SyncPair{T1: t.ID, T2: p.ID, Idx: i, T2Idx: j, Sync: true})
							return sync
						}
					}
				}
			} else if _, ok := m.ClosedChans[op.Ch]; ok && len(close) == 0 { //operation on a closed channel
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, Closed: true, Idx: i})
			}
		}
	}
	if len(async) > 0 {
		return async
	} else if len(close) > 0 {
		return close
	}
	return dataAccess
}

//GetNextActionWCommLink determines the next event with priority to sync operations. Memory accesses are delayed on purpose
func (m *Machine) GetNextActionWCommLink() (ret []SyncPair) {
	var async []SyncPair
	var sync []SyncPair
	var dataAccess []SyncPair
	var close []SyncPair
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek() //pre

		for i, op := range e.Ops {
			if op.Kind&util.CLS > 0 && len(close) == 0 {
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, DoClose: true, Idx: i})
			} else if op.BufSize > 0 && len(async) == 0 { //async op
				if op.Kind == util.PREPARE|util.SEND {
					c := m.AsyncChans2[op.Ch]
					if op.Mutex == util.RLOCK {
						// The lock must either be empty (c.Count < c.BufSize) or the current holder must be a reader
						// writer handling is sufficient with the empty lock conditions since writers are only allowed
						// if nobody else is using the lock
						if c.Count < 2 {
							async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncSend: true, IsSelect: len(e.Ops) > 1})
						}
					} else if op.Mutex == util.LOCK {
						if c.Count == 0 {
							async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncSend: true, IsSelect: len(e.Ops) > 1})
						}
					} else if c.Count < c.BufSize {
						async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncSend: true, IsSelect: len(e.Ops) > 1})
					}

				} else if op.Kind == util.PREPARE|util.RCV {
					c := m.AsyncChans2[op.Ch]
					if op.Mutex == util.RUNLOCK {
						if c.Count < 2 {
							async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true, IsSelect: len(e.Ops) > 1})
						}
					} else if op.Mutex == util.UNLOCK {
						if c.Count == 2 {
							async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true, IsSelect: len(e.Ops) > 1})
						}
					} else if c.Count > 0 {
						async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true, IsSelect: len(e.Ops) > 1})
					}
					// if c.Count > 0 {
					// 	async = append(async, SyncPair{T1: t.ID, T2: op.Ch, AsyncRcv: true, IsSelect: len(e.Ops) > 1})
					// }
				}
			} else if (op.Kind&util.WRITE > 0 || op.Kind&util.READ > 0) && len(dataAccess) == 0 { //data access
				dataAccess = append(dataAccess, SyncPair{T1: t.ID, T2: op.Ch, DataAccess: true, Idx: i})
			} else if op.Kind&util.CLS > 0 && len(close) == 0 { //channel closing
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, DoClose: true, Idx: i})
			} else if _, ok := m.ClosedChans[op.Ch]; ok {
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, Closed: true, Idx: i, IsSelect: len(e.Ops) > 1})
			} else if op.Kind&util.RCV > 0 {
				if len(t.Events) == 1 {
					// pending rcv, executed during the post processing
					continue
				}
				partner := t.Events[1].Partner

				for _, p := range m.Threads {
					if p.ID == t.ID || len(p.Events) == 0 || partner != p.ID {
						continue
					}
					pe := p.Peek()
					for j, pop := range pe.Ops {
						if pop.Ch == op.Ch && pop.Kind == util.PREPARE|util.SEND {
							sync = append(sync, SyncPair{T1: t.ID, T2: p.ID, Idx: i, T2Idx: j, Sync: true, IsSelect: len(e.Ops) > 1})
							return sync
						}
					}
				}
			} else if _, ok := m.ClosedChans[op.Ch]; ok && len(close) == 0 { //operation on a closed channel
				close = append(close, SyncPair{T1: t.ID, T2: op.Ch, Closed: true, Idx: i, IsSelect: len(e.Ops) > 0})
			}
		}
	}
	if len(async) > 0 {
		return async
	} else if len(close) > 0 {
		return close
	}
	return dataAccess
}

func (m *Machine) GetNextRandomActionWCommLink() (ret []SyncPair) {
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		e := t.Peek() //pre

		for i, op := range e.Ops {
			if op.Kind&util.CLS > 0 {
				ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch,
					DoClose: true, Idx: i})
			} else if op.Kind&util.WRITE > 0 || op.Kind&util.READ > 0 {
				ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, DataAccess: true,
					Idx: i})
			} else if op.BufSize > 0 {
				if op.Kind == util.PREPARE|util.SEND {
					c := m.AsyncChans2[op.Ch]
					if c.Count < c.BufSize {
						ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch,
							AsyncSend: true, IsSelect: len(e.Ops) > 1, Idx: i})
					}
				} else if op.Kind == util.PREPARE|util.RCV {
					c := m.AsyncChans2[op.Ch]
					if c.Count > 0 {
						ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch,
							AsyncRcv: true, IsSelect: len(e.Ops) > 1, Idx: i})
					}
				}
			} else if _, ok := m.ClosedChans[op.Ch]; ok {
				ret = append(ret, SyncPair{T1: t.ID, T2: op.Ch, Closed: true,
					IsSelect: len(e.Ops) > 1, Idx: i})
			} else if op.Kind&util.RCV > 0 {
				if len(t.Events) == 1 {
					// pending rcv, executed during the post processing
					continue
				}
				partner := t.Events[1].Partner

				for _, p := range m.Threads {
					if p.ID == t.ID || len(p.Events) == 0 || partner != p.ID {
						continue
					}
					pe := p.Peek()
					for j, pop := range pe.Ops {
						if pop.Ch == op.Ch && pop.Kind == util.PREPARE|util.SEND {
							ret = append(ret, SyncPair{T1: t.ID, T2: p.ID,
								Idx: i, T2Idx: j, Sync: true, IsSelect: len(e.Ops) > 1})
						}
					}
				}
			}
			if len(ret) > 0 {
				return
			}
		}
	}

	return
}

func (m *Machine) UpdateChanVc(p *SyncPair) {
	//	fmt.Println(">>", p.String())
	thread1 := m.Threads[p.T1]

	if p.AsyncSend {
		chanState := m.ChanVC[p.T2]
		if chanState == nil {
			chanState = util.NewChanState()
		}

		//prevReadCtxt := chanState.RContext

		if chanState.Wvc.Less(thread1.VC) {
			nVC := util.NewVC()
			nVC.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			chanState.Wvc = nVC
			//chanState.WContext = []string{thread1.ShortString()}
			chanState.WContext = make(map[string]string)
			chanState.WContext[thread1.ID] = thread1.ShortString()
		} else {
			chanState.Wvc.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			//chanState.WContext = append(chanState.WContext, thread1.ShortString())
			chanState.WContext[thread1.ID] = thread1.ShortString()
		}

		if len(chanState.Wvc) > 1 {
			s := fmt.Sprintf("%v", thread1.ShortString())
			//s = string(s[4 : len(s)-3])
			for _, v := range chanState.WContext {
				report.Alternative(s, v)
				//	fmt.Println(v)
			}
		}
		m.ChanVC[p.T2] = chanState

	} else if p.AsyncRcv {
		chanState := m.ChanVC[p.T2]
		if chanState == nil {
			chanState = util.NewChanState()
		}

		if chanState.Rvc.Less(thread1.VC) {
			nVC := util.NewVC()
			nVC.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			chanState.Rvc = nVC
			//chanState.RContext = []string{thread1.ShortString()}
			chanState.RContext = make(map[string]string)
			chanState.RContext[thread1.ID] = thread1.ShortString()
		} else {
			chanState.Rvc.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			//chanState.RContext = append(chanState.RContext, thread1.ShortString())
			chanState.RContext[thread1.ID] = thread1.ShortString()
		}

		if len(chanState.Rvc) > 1 {
			s := fmt.Sprintf("%v", thread1.ShortString())
			for _, v := range chanState.RContext {
				report.Alternative(s, v)
			}
		}

		m.ChanVC[p.T2] = chanState

	} else if p.Sync {
		thread2 := m.Threads[p.T2]
		chanState := m.ChanVC[thread1.Events[0].Ops[0].Ch]
		if chanState == nil {
			chanState = util.NewChanState()
		}
		//prevReadCtxt := chanState.RContext
		//prevWriteCtxt := chanState.WContext
		if chanState.Rvc.Less(thread1.VC) {
			nVC := util.NewVC()
			nVC.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			chanState.Rvc = nVC
			//	prevReadCtxt = chanState.RContext
			//chanState.RContext = []string{thread1.ShortString()}
			chanState.RContext = make(map[string]string)
			chanState.RContext[thread1.ID] = thread1.ShortString()
		} else {
			chanState.Rvc.AddEpoch(util.NewEpoch(thread1.ID, thread1.VC[thread1.ID]))
			//chanState.RContext = append(chanState.RContext, thread1.ShortString())
			chanState.RContext[thread1.ID] = thread1.ShortString()
		}

		if len(chanState.Rvc) > 1 {
			s := fmt.Sprintf("%v", thread1.ShortString())
			for _, v := range chanState.RContext {

				report.Alternative(s, v)
			}
		}

		if chanState.Wvc.Less(thread2.VC) {
			nVC := util.NewVC()
			nVC.AddEpoch(util.NewEpoch(thread2.ID, thread2.VC[thread2.ID]))
			chanState.Wvc = nVC
			//prevWriteCtxt = chanState.WContext
			//chanState.WContext = []string{thread2.ShortString()}
			chanState.WContext = make(map[string]string)
			chanState.WContext[thread2.ID] = thread2.ShortString()
		} else {
			chanState.Wvc.AddEpoch(util.NewEpoch(thread2.ID, thread2.VC[thread2.ID]))
			//chanState.WContext = append(chanState.WContext, thread2.ShortString())
			chanState.WContext[thread2.ID] = thread2.ShortString()
		}

		if len(chanState.Wvc) > 1 {
			s := fmt.Sprintf("%v", thread2.ShortString())
			for _, v := range chanState.WContext {

				report.Alternative(s, v)
			}
		}

		m.ChanVC[thread1.Events[0].Ops[0].Ch] = chanState
	} else if p.Closed && p.T2 == "0" {
		//default case of select

	}

	// fmt.Println("------------------------")
	// for k, v := range m.ChanVC {
	// 	fmt.Println(k)
	// 	fmt.Println(v.Wvc)
	// 	fmt.Println(v.Rvc)
	// 	fmt.Println(v.WContext, "||", v.RContext)
	// 	fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%")
	// }
}

func (m *Machine) PostProcessing() {
	for _, t := range m.Threads {
		if len(t.Events) == 0 {
			continue
		}
		for _, e := range t.Events {
			fmt.Println("pending events", t.ID, len(t.Events), e.String())
		}
	}
	fmt.Printf("\n\n")
	for {
		done := 0

		for _, t := range m.Threads {
			if len(t.Events) == 0 {
				done++
				continue
			}

			e := t.Peek()
			op := e.Ops[0]

			t.VC.Add(t.ID, 1)
			e.VC = t.VC.Clone()

			//handle close postprocessing
			if op.Kind == util.PREPARE|util.SEND {
				if _, ok := m.ClosedChans[op.Ch]; ok {
					//snd after close report error
					//time doesn't matter here because the close already happened
					//so the send is either after the close or concurrent but not before
					// / there is a schedule where its after the close. Either report error
					fmt.Println("Send after/concurrent close op")
				}
			}

			//handle select stmts + concurrent read writes
			if op.Kind == util.PREPARE|util.SEND || op.Kind == util.PREPARE|util.RCV {
				report.AddEvent(e, nil)
				for _, s := range m.Selects {
					for _, x := range s.Ev.Ops {
						if op.Ch == x.Ch { //channel op on the same channel as a case in this select
							//check vector clocks. if the threads vc  is before or concurrent to the select
							// report alternative case for this select
							dual := util.OpKind(util.PREPARE | util.SEND)
							if x.Kind == util.PREPARE|util.SEND {
								dual = util.PREPARE | util.RCV
							}
							if !s.VC.Less(t.VC) && dual == e.Ops[0].Kind {
								tmp := fmt.Sprintf("%v", s.Ev)
								report.SelectAlternative(tmp, e.String())
								//fmt.Printf("1Event %v is an alternative for select statement \n\t%v\n", e, s.Ev)
							}
						}
					}
				}

				chanState := m.ChanVC[op.Ch]
				if chanState == nil {
					//continue
				} else {

					if op.Kind&util.SEND > 0 {
						if !chanState.Wvc.Less(t.VC) {
							chanState.Wvc.AddEpoch(util.NewEpoch(t.ID, t.VC[t.ID]))
							//	chanState.WContext = append(chanState.WContext, t.ShortString())
							chanState.WContext[t.ID] = t.ShortString()
						}
						if len(chanState.Wvc) > 1 {
							s := fmt.Sprintf("%v", t.ShortString())
							for _, v := range chanState.WContext {
								report.Alternative(s, v)
							}
						}
					} else if op.Kind&util.RCV > 0 {
						if !chanState.Rvc.Less(t.VC) {
							chanState.Rvc.AddEpoch(util.NewEpoch(t.ID, t.VC[t.ID]))
							//	chanState.RContext = append(chanState.RContext, t.ShortString())
							chanState.RContext[t.ID] = t.ShortString()
						}
						if len(chanState.Rvc) > 1 {
							s := fmt.Sprintf("%v", t.ShortString())
							for _, v := range chanState.RContext {
								report.Alternative(s, v)
							}
						}
					}
					m.ChanVC[op.Ch] = chanState
				}
			}

			t.Pop()
			m.Threads[t.ID] = t
		}

		if done == len(m.Threads) {
			return
		}
	}
}
