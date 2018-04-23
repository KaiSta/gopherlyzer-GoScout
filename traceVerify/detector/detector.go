package race

import (
	"fmt"
	"time"

	"../util"
	"./traceReplay"
	"github.com/fatih/color"
)

func createThreads(items []util.Item) map[string]util.Thread {
	tmap := make(map[string]util.Thread)
	for i, item := range items {
		t, ok := tmap[item.Thread]
		if !ok {
			t.ID = item.Thread
			t.VC = util.NewVC()
			t.RVC = util.NewVC()
		}
		t.Events = append(t.Events, &items[i])

		tmap[item.Thread] = t
	}
	return tmap
}

//maybe deprecated not used so far
func getAsyncChans(items []util.Item) map[string][]util.Item {
	chans := make(map[string][]util.Item)
	for _, i := range items {
		for _, o := range i.Ops {
			if o.BufSize == 0 {
				continue
			}
			_, ok := chans[o.Ch]
			if !ok {
				chans[o.Ch] = []util.Item{}
			}
		}
	}
	return chans
}

func getAsyncChans2(items []util.Item) map[string]traceReplay.AsyncChan {
	chans := make(map[string]traceReplay.AsyncChan)
	for _, i := range items {
		for _, o := range i.Ops {
			if o.BufSize == 0 {
				continue
			}
			//extend here to add it as special channel for rw locks
			if o.Mutex == 0 {
				if _, ok := chans[o.Ch]; !ok {
					achan := traceReplay.AsyncChan{BufSize: o.BufSize, Buf: make([]traceReplay.BufField, o.BufSize)}
					for i := range achan.Buf {
						achan.Buf[i].VC = util.NewVC()
					}
					chans[o.Ch] = achan
				}
			} else {
				if _, ok := chans[o.Ch]; !ok {
					achan := traceReplay.AsyncChan{BufSize: 2, Buf: make([]traceReplay.BufField, 2)}
					for i := range achan.Buf {
						achan.Buf[i].VC = util.NewVC()
					}
					if o.Mutex == util.RLOCK {
						achan.IsRWLock = true
					}
					chans[o.Ch] = achan
				} else if o.Mutex == util.RLOCK {
					c := chans[o.Ch]
					c.IsRWLock = true
					chans[o.Ch] = c
				}
			}
		}
	}
	return chans
}

// func eraser(v *util.VarState3, t *util.Thread, ev *util.Item) {
// 	if v.MutexSet == nil {
// 		v.MutexSet = make(map[string]struct{})
// 		for k, val := range t.MutexSet {
// 			v.MutexSet[k] = val
// 		}
// 		return
// 	}
// 	for k := range v.MutexSet {
// 		_, ok := t.MutexSet[k]
// 		if !ok {
// 			delete(v.MutexSet, k)
// 		}
// 	}
// 	if len(v.MutexSet) == 0 && v.State == util.SHARED {
// 		fmt.Printf("Eraser: possible data race on %v\n@%v\n\n", ev.Ops[0].Ch, ev)
// 	}
// }

func replayWithVCs(m *traceReplay.Machine, handleRW func(*traceReplay.Machine), jsonFlag, plain, bench bool) {
	for {
		m.StartAllthreads()
		handleRW(m)

		pairs := m.GetSyncPairs()
		pairs = append(pairs, m.GetAsyncActions()...)

		if len(pairs) == 0 {
			break
		}

		for _, p := range pairs {
			if p.AsyncSend {
				t1 := m.Threads[p.T1]
				ch := m.AsyncChans2[p.T2]

				if ch.Count == ch.BufSize {
					continue
				}

				t1.VC.Add(t1.ID, 2)
				ch.Buf[ch.Next].Item = t1.Events[0]
				t1.VC.Sync(ch.Buf[ch.Next].VC)

				ch.Next++
				if ch.Next >= ch.BufSize {
					ch.Next = 0
				}
				ch.Count++

				t1.Pop()
				t1.Pop()

				m.Threads[t1.ID] = t1
				m.AsyncChans2[p.T2] = ch
				// t1.Events[0].VC = t1.VC.Clone() //necessary, event is put into chan buffer, receiver takes it out and syncs it vc with the stored vc that is cloned here.
				// list := m.AsyncChans[t1.Events[1].Ops[0].Ch]
				// list = append(list, t1.Events[0])
				// m.AsyncChans[t1.Events[1].Ops[0].Ch] = list
				//	continue
			} else if p.AsyncRcv {
				ch := m.AsyncChans2[p.T2]

				if ch.Count > 0 {
					t1 := m.Threads[p.T1]
					t1.VC.Add(t1.ID, 2)
					t1.VC.Sync(ch.Buf[ch.Rnext].VC)
					ch.Buf[ch.Rnext].Item = nil

					ch.Rnext++
					if ch.Rnext >= ch.BufSize {
						ch.Rnext = 0
					}
					ch.Count--

					t1.Pop()
					t1.Pop()

					m.Threads[t1.ID] = t1
					m.AsyncChans2[p.T2] = ch
				}
				// list := m.AsyncChans[p.T2]
				// if len(list) > 0 {
				// 	partner := list[0]
				// 	t1 := m.Threads[p.T1]

				// 	if partner.Thread != t1.Events[1].Partner {
				// 		continue
				// 	}

				// 	if len(list) > 1 {
				// 		list = list[1:]
				// 	} else {
				// 		list = []util.Item{}
				// 	}
				// 	m.AsyncChans[p.T2] = list

				// 	t1.VC.Add(t1.ID, 2)
				// 	t1.VC.Sync(partner.VC)

				// 	t1.Pop()
				// 	t1.Pop()
				// 	m.Threads[t1.ID] = t1
				// }
				//	continue
			} else if p.DoClose {
				t1 := m.Threads[p.T1]
				t1.VC.Add(t1.ID, 2)
				m.ClosedChansVC[p.T2] = t1.VC.Clone()
				m.ClosedChans[p.T2] = struct{}{}
				t1.Pop()
				t1.Pop()
				m.Threads[t1.ID] = t1
				//continue
			} else if p.Closed {
				t1 := m.Threads[p.T1]
				t1.VC.Add(t1.ID, 2)
				t1.Pop()
				t1.Pop()
				m.Threads[t1.ID] = t1
				//continue
			} else {
				t1 := m.Threads[p.T1]
				t2 := m.Threads[p.T2]
				if len(t1.Events) > 1 {
					if t1.Events[1].Partner != t2.ID {
						continue
					}

					//prepare + commit
					t1.VC.Add(t1.ID, 2)
					t2.VC.Add(t2.ID, 2)
					t1.VC.Sync(t2.VC)
					t2.VC.Sync(t1.VC)

					//add 'mutex' to the mutex set of the thread that executed a send
					if t1.Events[0].Ops[0].Mutex&util.LOCK > 0 {
						if t1.Events[0].Ops[0].Kind&util.SEND > 0 {
							if t1.MutexSet == nil {
								t1.MutexSet = make(map[string]struct{})
							}
							t1.MutexSet[t2.ID] = struct{}{}
						} else {
							if t2.MutexSet == nil {
								t2.MutexSet = make(map[string]struct{})
							}
							t2.MutexSet[t1.ID] = struct{}{}
						}
					} else if t1.Events[0].Ops[0].Mutex&util.UNLOCK > 0 {
						if t1.Events[0].Ops[0].Kind&util.SEND > 0 {
							delete(t1.MutexSet, t2.ID)
						} else {
							delete(t2.MutexSet, t1.ID)
						}
					}

					t1.Pop()
					t1.Pop()
					t2.Pop()
					t2.Pop()
					m.Threads[t1.ID] = t1
					m.Threads[t2.ID] = t2
				} else {
					if len(t1.Events) > 0 {
						t1.VC.Add(t1.ID, 1)
						t1.Pop()
						m.Threads[p.T1] = t1
					}
				}
			}
		}
	}

	// //really necessary?
	// for _, t := range m.Threads {
	// 	for {
	// 		if len(t.Events) > 0 && t.Peek().Ops[0].Kind&util.WAIT == 0 {
	// 			if t.Peek().Ops[0].Kind&util.SIG > 0 {
	// 				t.Pop()
	// 				m.Threads[t.ID] = t
	// 			} else {
	// 				t.VC.Add(t.ID, 1)
	// 				t.Events[0].VC = t.VC.Clone()
	// 				// items = append(items, t.Peek())
	// 				t.Pop()
	// 				m.Threads[t.ID] = t
	// 			}
	// 		} else {
	// 			break
	// 		}
	// 	}
	// }

}

func reconstructVCs(m *traceReplay.Machine, jsonFlag, plain, bench bool) []*util.Item {
	var items []*util.Item

	for {
		m.StartAllthreads()
		items = append(items, m.ExAllRW()...)

		pairs := m.GetSyncPairs()

		pairs = append(pairs, m.GetAsyncActions()...)

		if len(pairs) == 0 {
			break
		}

		for _, p := range pairs {
			if p.AsyncSend {
				t1 := m.Threads[p.T1]
				t1.VC.Add(t1.ID, 1)
				t1.Events[0].VC = t1.VC.Clone()
				t1.VC.Add(t1.ID, 1)
				t1.Events[1].VC = t1.VC.Clone()
				list := m.AsyncChans[t1.Events[1].Ops[0].Ch]
				list = append(list, t1.Events[0])
				m.AsyncChans[t1.Events[0].Ops[0].Ch] = list

				// items = append(items, t1.Events[0])
				// items = append(items, t1.Events[1])
				t1.Pop()
				t1.Pop()
				m.Threads[t1.ID] = t1
				continue
			} else if p.AsyncRcv {
				list := m.AsyncChans[p.T2]
				if len(list) > 0 {
					partner := list[0]
					t1 := m.Threads[p.T1]

					if partner.Thread != t1.Events[1].Partner {
						continue
					}

					if len(list) > 1 {
						list = list[1:]
					} else {
						list = []*util.Item{}
					}
					m.AsyncChans[p.T2] = list

					t1.VC.Add(t1.ID, 1)
					t1.Events[0].VC = t1.VC.Clone()
					t1.VC.Add(t1.ID, 1)
					t1.VC.Sync(partner.VC)
					t1.Events[1].VC = t1.VC.Clone()
					// items = append(items, t1.Events[0])
					// items = append(items, t1.Events[1])

					//chosenPartner[t1.events[0].ShortString()] = partner
					//	chosenPartner[partner.ShortString()] = t1.events[0]

					t1.Pop()
					t1.Pop()
					m.Threads[t1.ID] = t1
				}
				continue
			}

			if p.DoClose {
				t1 := m.Threads[p.T1]
				t1.VC.Add(t1.ID, 1)
				t1.Events[0].VC = t1.VC.Clone()
				t1.VC.Add(t1.ID, 1)
				t1.Events[1].VC = t1.VC.Clone()
				m.ClosedChansVC[p.T2] = t1.VC.Clone()
				m.ClosedChans[p.T2] = struct{}{}
				// items = append(items, t1.Events[0])
				// items = append(items, t1.Events[1])
				t1.Pop()
				t1.Pop()
				m.Threads[t1.ID] = t1
				continue
			}

			if p.Closed {
				t1 := m.Threads[p.T1]
				t1.VC.Add(t1.ID, 1)
				t1.Events[0].VC = t1.VC.Clone()
				t1.VC.Add(t1.ID, 1)
				t1.Events[1].VC = t1.VC.Clone()
				// items = append(items, t1.Events[0])
				// items = append(items, t1.Events[1])
				t1.Pop()
				t1.Pop()
				m.Threads[t1.ID] = t1
				continue
			}
			t1 := m.Threads[p.T1]
			t2 := m.Threads[p.T2]
			if len(t1.Events) > 1 {
				if t1.Events[1].Partner != t2.ID {
					continue
				}

				//prepare
				t1.VC.Add(t1.ID, 1)
				t1.Events[0].VC = t1.VC.Clone()
				//fmt.Println(t1.ID, t1.Events[0], t2.ID, t2.vc)
				t2.VC.Add(t2.ID, 1)
				t2.Events[0].VC = t2.VC.Clone()

				//commit
				t1.VC.Add(t1.ID, 1)
				t2.VC.Add(t2.ID, 1)
				t1.VC.Sync(t2.VC)
				t2.VC.Sync(t1.VC)
				t1.Events[1].VC = t1.VC.Clone()
				t2.Events[1].VC = t2.VC.Clone()

				//	chosenPartner[t1.events[0].ShortString()] = t2.peek()
				//	chosenPartner[t2.events[0].ShortString()] = t1.peek()

				// items = append(items, t1.Events[0])
				// items = append(items, t1.Events[1])
				// items = append(items, t2.Events[0])
				// items = append(items, t2.Events[1])
				t1.Pop()
				t1.Pop()
				t2.Pop()
				t2.Pop()
				m.Threads[t1.ID] = t1
				m.Threads[t2.ID] = t2
			} else {
				if len(t1.Events) > 0 {
					t1.VC.Add(t1.ID, 1)
					t1.Events[0].VC = t1.VC.Clone()
					//items = append(items, t1.Events[0])
					t1.Pop()
					m.Threads[p.T1] = t1
				}

			}
		}
	}
	for _, t := range m.Threads {
		for {
			if len(t.Events) > 0 && t.Peek().Ops[0].Kind&util.WAIT == 0 {
				if t.Peek().Ops[0].Kind&util.SIG > 0 {
					t.Pop()
					m.Threads[t.ID] = t
				} else {
					t.VC.Add(t.ID, 1)
					t.Events[0].VC = t.VC.Clone()
					// items = append(items, t.Peek())
					t.Pop()
					m.Threads[t.ID] = t
				}
			} else {
				break
			}
		}

	}
	//fmt.Println("Prep2Time:", time.Since(s3))
	//	fmt.Println(len(items))
	return items
}

func findAlternatives(items []util.Item, plain, json, bench bool) {
	s4 := time.Now()
	cache := make(map[struct {
		ch string
		op util.OpKind
	}][]*util.Item)

	//fill cache in advance
	for i, it := range items {
		for _, op := range it.Ops {
			s := struct {
				ch string
				op util.OpKind
			}{op.Ch, op.Kind}
			tmp := cache[s]
			tmp = append(tmp, &items[i])
			cache[s] = tmp
		}
	}

	usedMap := make(map[string]string)
	for k, v := range cache {
		if k.op&util.COMMIT > 0 {
			for _, x := range v {
				if len(x.Partner) > 0 { //rcv commit
					usedMap[x.Ops[0].SourceRef+x.Thread] = x.Partner
					usedMap[x.Partner] = x.Ops[0].SourceRef + x.Thread
				}
			}
		}
	}

	var alternatives []util.Alternative2

	//alternatives := make(map[string]Alternative2)
	for k, v := range cache {
		if k.op&util.PREPARE > 0 {
			opKind := util.OpKind(0)
			if k.op == util.PREPARE|util.RCV {
				opKind = util.PREPARE | util.SEND
			} else if k.op == util.PREPARE|util.SEND {
				opKind = util.PREPARE | util.RCV
			}
			s := struct {
				ch string
				op util.OpKind
			}{k.ch, opKind}
			partners := cache[s]

			for _, x := range v {
				alt := util.Alternative2{Op: x, Unused: make([]*util.Item, 0), Used: make([]*util.Item, 0)}
				//	fmt.Println("alternatives for", x.ShortString())
				for _, y := range partners {
					if x.Thread == y.Thread {
						continue
					}

					if !x.VC.Less(y.VC) && !y.VC.Less(x.VC) {
						used := false

						if usedMap[x.Thread] == y.Ops[0].SourceRef+y.Thread || usedMap[y.Thread] == x.Ops[0].SourceRef+x.Thread {
							used = true
						}
						if usedMap[x.Ops[0].SourceRef+x.Thread] == y.Thread || usedMap[y.Ops[0].SourceRef+y.Thread] == x.Thread {
							used = true
						}

						if used {
							alt.Used = append(alt.Used, y)
						} else {
							alt.Unused = append(alt.Unused, y)
						}

					}
				}
				if len(alt.Unused) > 0 || len(alt.Used) > 0 {
					alternatives = append(alternatives, alt)
					//alternatives[alt.Op.ShortString()] = alt
				}
			}
		}
	}

	fmt.Println("found", len(alternatives), "alternatives")

	if plain {
		for _, x := range alternatives {
			fmt.Println("alternatives for", x.Op)
			for i := range x.Unused {
				color.HiRed(fmt.Sprintf("\t%v", x.Unused[i].ShortString()))
			}
			for i := range x.Used {
				color.HiGreen(fmt.Sprintf("\t%v", x.Used[i].ShortString()))
			}
		}
	}
	fmt.Println("AlternativeSearchTime:", time.Since(s4))

	if bench || json {
		return
	}

	for k, v := range cache {
		if k.op != util.COMMIT|util.CLS {
			continue
		}
		for _, c := range v {
			fmt.Println("parallel to", c)
			//rcv partner
			s := struct {
				ch string
				op util.OpKind
			}{k.ch, util.PREPARE | util.RCV}
			rcvpartners := cache[s]
			for _, x := range rcvpartners {
				if !x.VC.Less(c.VC) && !c.VC.Less(x.VC) {
					color.HiGreen("\t%v", x.ShortString())
				}
			}

			//snd partner
			s = struct {
				ch string
				op util.OpKind
			}{k.ch, util.PREPARE | util.SEND}
			sndpartners := cache[s]
			for _, x := range sndpartners {
				if !x.VC.Less(c.VC) && !c.VC.Less(x.VC) {
					color.HiRed("\t%v", x.ShortString())
				}
			}
		}

	}
}

func findRaces(items []util.Item) {
	for i := 0; i < len(items); i++ {
		if items[i].Ops[0].Kind == util.COMMIT|util.READ {
			continue
		}
		vc := items[i].VC
		for j := i + 1; j < len(items); j++ {
			if items[j].Ops[0].Ch != items[i].Ops[0].Ch {
				continue
			}
			if !(vc.Less(items[j].VC)) && !(items[j].VC.Less(vc)) {
				fmt.Println("Race between", items[i], items[j])
			}
		}
	}
}

func Run(tracePath string, json, plain, bench bool) {
	// items := parser.ParseTrace(tracePath)

	// threads := createThreads(items)
	// //	aChans := getAsyncChans(items)
	// aChans2 := getAsyncChans2(items)
	// //	fmt.Println(aChans2)

	// closed := make(map[string]struct{})
	// closed["0"] = struct{}{}

	// m1 := &Machine{threads, aChans, closed, make(map[string]util.VectorClock), false, make(map[string]*util.VarState1), nil, nil}
	// replayWithVCs(m1, exAllRWVC, json, plain, bench)

	// m2 := &Machine{threads, aChans, closed, make(map[string]util.VectorClock), false, nil, make(map[string]*util.VarState2), nil}
	// replayWithVCs(m2, exAllRWEpoch, json, plain, bench)

	//	m3 := &traceReplay.Machine{threads, nil, closed, make(map[string]util.VectorClock),
	//		false, nil, nil, make(map[string]*util.VarState3), aChans2, nil,
	//		make([]traceReplay.SelectStore, 0)}
	//replayWithVCs(m3, traceReplay.ExAllRWEpochAndEraser2, json, plain, bench)
	// res := reconstructVCs(&Machine{threads, aChans, closed, make(map[string]util.VectorClock), false, nil, nil, nil}, json, plain, bench)
	// findRaces(res)
}
