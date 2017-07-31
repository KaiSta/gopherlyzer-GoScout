package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const (
	PREPARE   = 1 << iota
	COMMIT    = 1 << iota
	SEND      = 1 << iota
	RCV       = 1 << iota
	CLS       = 1 << iota
	RMVSEND   = 0xF ^ SEND
	RMVRCV    = 0xF ^ RCV
	RMVPREP   = 0xF ^ PREPARE
	RMVCOT    = 0xF ^ COMMIT
	NOPARTNER = "-"
)

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

type OpKind int

func (k OpKind) String() string {
	sym := "!"
	stat := "P"

	if k&RCV > 0 {
		sym = "?"
	} else if k&CLS > 0 {
		sym = "#"
	}
	if k&COMMIT > 0 {
		stat = "C"
	}
	return fmt.Sprintf("%v,%v", sym, stat)
}

type Operation struct {
	ch        string
	kind      OpKind
	bufSize   uint64
	sourceRef string
}

func (o Operation) String() string {
	return fmt.Sprintf("%v(%v),%v,%v", o.ch, o.bufSize, o.kind, o.sourceRef)
}

type Item struct {
	thread  string
	ops     []Operation
	partner string
	vc      VectorClock
}

func (o Item) clone() Item {
	var ops []Operation
	for _, x := range o.ops {
		ops = append(ops, x)
	}
	vc := NewVC()
	for k, v := range o.vc {
		vc[k] = v
	}
	return Item{o.thread, ops, o.partner, vc}
}

func (o Item) String() string {
	var ops string
	for i, p := range o.ops {
		ops += fmt.Sprintf("(%v)", p)

		if i+1 < len(o.ops) {
			ops += ","
		}
	}

	if o.partner != "" {
		return fmt.Sprintf("%v,[%v],%v", o.thread, ops, o.partner)
	}
	return fmt.Sprintf("%v,[%v]", o.thread, ops)
}
func (o Item) ShortString() string {
	var ops string
	for i, p := range o.ops {
		ops += fmt.Sprintf("(%v)", p)

		if i+1 < len(o.ops) {
			ops += ","
		}
	}

	if o.partner != "" {
		return fmt.Sprintf("%v,[%v],%v", o.thread, ops, o.partner)
	}
	return fmt.Sprintf("%v,[%v]", o.thread, ops)
}

func getTName(l string) (string, string) {
	var name string
	i := 0
	for _, c := range l {
		i++
		if c == ',' {
			break
		}

		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			name += string(c)
		}
	}
	return name, l[i:]
}

func getOps(l string) ([]Operation, string) {
	var ops []Operation
	s := 0
	i := 0
	var curr Operation
	var currBufSize string
	stop := false
	isCommit := false
	for _, c := range l {
		if stop {
			break
		}
		i++
		switch s {
		case 0:
			if c == '[' {
				s++
			} else {
				panic("invalid trace format")
			}
		case 1:
			if c == '(' {
				curr = Operation{}
				s++
			} else {
				panic("invalid trace format")
			}
		case 2:
			if c == ',' {
				s++
			} else {
				curr.ch += string(c)
			}
		case 3:
			if c == ',' {
				bufsize, _ := strconv.ParseUint(currBufSize, 10, 64)
				curr.bufSize = bufsize
				currBufSize = ""
				s++
			} else {
				currBufSize += string(c)
			}
		case 4:
			if c == '!' {
				curr.kind |= SEND
			} else if c == '?' {
				curr.kind |= RCV
			} else if c == '#' {
				curr.kind |= CLS
			}
			if c == ',' {
				s++
			}

		case 5:
			if c == ')' {
				ops = append(ops, curr)
				s++
			} else {
				curr.sourceRef += string(c)
			}
		case 6:
			if c == ',' {
				s = 1
			} else if c == ']' {
				s++
			}
		case 7:
			if c == 'C' {
				isCommit = true
			}
			if c != ',' {
				stop = true
			}

		}
	}
	for i := range ops {
		if isCommit {
			ops[i].kind |= COMMIT
		} else {
			ops[i].kind |= PREPARE
		}
	}
	i++
	return ops, l[i:]
}

func parseTrace(s string) []Item {
	data, err := ioutil.ReadFile(s)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(data), "\n")

	var items []Item
	for _, l := range lines {
		if len(l) == 0 || len(l) == 1 {
			break
		}
		var item Item
		var rest string
		item.thread, rest = getTName(l)
		item.ops, rest = getOps(rest)
		item.partner, rest = getTName(rest)

		items = append(items, item)
	}

	return items
}

type VectorClock map[string]int

func (vc VectorClock) String() string {
	s := "["
	for k, v := range vc {
		s += fmt.Sprintf("(%v,%v)", k, v)
	}
	s += "]"
	return s
}
func (vc VectorClock) sync(pvc VectorClock) {
	for k, v := range vc {
		pv := pvc[k]
		vc[k] = max(v, pv)
	}
	for k, v := range pvc {
		pv := vc[k]
		vc[k] = max(v, pv)
	}
}
func (vc VectorClock) Add(k string, val int) {
	v := vc[k]
	v += val
	vc[k] = v
}
func (vc VectorClock) less(pvc VectorClock) bool {
	f := false
	for k := range vc {
		if vc[k] > pvc[k] {
			return false
		}
		if vc[k] < pvc[k] {
			f = true
		}
	}
	for k := range pvc {
		if vc[k] > pvc[k] {
			return false
		}
		if vc[k] < pvc[k] {
			f = true
		}
	}
	return f
}
func (vc VectorClock) clone() VectorClock {
	nvc := NewVC()
	for k, v := range vc {
		nvc[k] = v
	}
	return nvc
}

func NewVC() VectorClock {
	return make(VectorClock)
}

type Thread struct {
	ID     string
	events []Item
	vc     VectorClock
}

func (t Thread) String() string {
	return fmt.Sprintf("(%v, %v)", t.ID, t.events)
}

func (t Thread) clone() Thread {
	var items []Item
	for _, i := range t.events {
		items = append(items, i)
	}
	vc := NewVC()
	for k, v := range t.vc {
		vc[k] = v
	}
	return Thread{t.ID, items, vc}
	//	return Thread{t.ID, t.isBlocked, t.done, items, t.systemState}
}

func (t Thread) peek() Item {
	return t.events[0]
}
func (t *Thread) pop() {
	if len(t.events) > 1 {
		t.events = t.events[1:]
	} else {
		t.events = []Item{}
	}
}

func createThreads(items []Item) map[string]Thread {
	tmap := make(map[string]Thread)
	for _, item := range items {
		t := tmap[item.thread]
		t.ID = item.thread
		t.events = append(t.events, item)
		t.vc = NewVC()
		tmap[item.thread] = t
	}
	return tmap
}

type machine struct {
	threads       map[string]Thread
	asyncChans    map[string][]Item
	closedChans   map[string]struct{}
	closedChansVC map[string]VectorClock
	stopped       bool
}

func (m machine) clone() machine {
	threads := make(map[string]Thread)
	for k, v := range m.threads {
		threads[k] = v.clone()
	}
	asyncChans := make(map[string][]Item)
	for k, v := range m.asyncChans {
		var ops []Item
		for _, i := range v {
			ops = append(ops, i.clone())
		}
		asyncChans[k] = ops
	}
	closedChans := make(map[string]struct{})
	for k, v := range m.closedChans {
		closedChans[k] = v
	}
	return machine{threads, asyncChans, closedChans, nil, false}
}

type syncPair struct {
	t1        string
	t2        string
	asyncSend bool
	asyncRcv  bool
	closed    bool
	doClose   bool
	idx       int
	t2Idx     int
}

func (m *machine) getSyncPairs() (ret []syncPair) {
	for _, t := range m.threads {
		if len(t.events) == 0 {
			continue
		}
		e := t.peek()
		for i, op := range e.ops {
			if op.kind&CLS > 0 {
				ret = append(ret, syncPair{t1: t.ID, t2: op.ch, doClose: true, idx: i})
				continue
			}
			if op.kind&RCV == 0 || op.bufSize > 0 {
				continue
			}
			if _, ok := m.closedChans[op.ch]; ok {
				//closed channel
				ret = append(ret, syncPair{t1: t.ID, t2: op.ch, closed: true, idx: i})
				continue
			}
			for _, p := range m.threads {
				if p.ID == t.ID || len(p.events) == 0 {
					continue
				}
				pe := p.peek()
				for j, pop := range pe.ops {
					if pop.ch == op.ch && pop.kind == OpKind(PREPARE|SEND) {
						ret = append(ret, syncPair{t1: t.ID, t2: p.ID, idx: i, t2Idx: j})
					}
				}
			}

		}
	}
	return
}
func (m *machine) getAsyncActions() (ret []syncPair) {
	for _, t := range m.threads {
		if len(t.events) == 0 {
			continue
		}
		e := t.peek()

		for _, op := range e.ops {
			if op.kind == OpKind(PREPARE|SEND) && op.bufSize > 0 {
				ret = append(ret, syncPair{t1: t.ID, t2: op.ch, asyncSend: true})
			} else if op.kind == OpKind(PREPARE|RCV) && op.bufSize > 0 {
				list := m.asyncChans[op.ch]
				if len(list) > 0 {
					ret = append(ret, syncPair{t1: t.ID, t2: op.ch, asyncRcv: true})
				}
			}
		}
	}
	return
}

func addSendAlts(possiblePaths map[string]map[string]struct {
	covered int
	loc     string
}) {
	sendMap := make(map[string]map[string]struct {
		covered int
		loc     string
	})

	for k, v := range possiblePaths {
		for _, v2 := range v {
			list := sendMap[v2.loc]
			if list == nil {
				list = make(map[string]struct {
					covered int
					loc     string
				})
			}
			list[k] = struct {
				covered int
				loc     string
			}{v2.covered, k}
			sendMap[v2.loc] = list
		}
	}
	for k, v := range sendMap {
		possiblePaths[k] = v
	}
}

func (m *machine) sync(p syncPair, possiblePaths map[string]map[string]struct {
	covered int
	loc     string
}) {
	t1 := m.threads[p.t1]
	t2 := m.threads[p.t2]

	if len(t1.events) > 1 {
		if t1.events[1].partner == t2.ID {
			prep := t1.peek().String()
			prep += fmt.Sprintf(",%v", p.idx)
			x := possiblePaths[prep]
			y := x[t2.ID]
			y.covered = 1
			x[t2.ID] = y
			possiblePaths[prep] = x
		}
		t1.pop() //PREP
		t1.pop() //COMMIT

		t2.pop() //PREP
		t2.pop() //COMMIT
		m.threads[p.t1] = t1
		m.threads[p.t2] = t2

	} else {
		color.HiCyan(fmt.Sprintf("Dangling prepare @%v", strings.TrimSpace(t1.events[0].String())))
		t1.pop()
		m.threads[p.t1] = t1
	}
}
func (m *machine) asyncAction(p syncPair, possiblePaths map[string]map[string]struct {
	covered int
	loc     string
}) {
	thread := m.threads[p.t1]
	ch := p.t2
	list := m.asyncChans[ch]
	if p.asyncSend {
		list = append(list, thread.peek())
	} else if p.asyncRcv {
		if len(m.threads[p.t1].events) > 1 {
			partner := list[0]
			if len(list) > 1 {
				list = list[1:]
			} else {
				list = []Item{}
			}

			if m.threads[p.t1].events[1].partner == partner.thread {
				prep := m.threads[p.t1].peek().String()
				x := possiblePaths[prep]
				y := x[partner.thread]
				y.covered = 1
				x[partner.thread] = y
				possiblePaths[prep] = x
			}
		} else {

			//	thread.pop()

		}
	}
	m.asyncChans[ch] = list

	thread.pop() //PREPARE
	thread.pop() //COMMIT
	m.threads[p.t1] = thread
}
func (m *machine) closedRcv(p syncPair, rcvOnClosed map[string]struct{}) {
	if p.closed {
		t := m.threads[p.t1]
		if p.t2 != "0" {
			rcvOnClosed[t.peek().String()] = struct{}{}
		}

		t.pop()
		t.pop()
		m.threads[p.t1] = t
	}
}

func (m *machine) closeChan(p syncPair) {
	t := m.threads[p.t1]
	t.pop()
	t.pop()
	m.threads[p.t1] = t
	m.closedChans[p.t2] = struct{}{}
}

func (m *machine) updatePossiblePaths(p syncPair, possiblePaths map[string]map[string]struct {
	covered int
	loc     string
}) {
	var partner Item
	if p.asyncRcv {
		list := m.asyncChans[p.t2]
		if len(list) > 0 {
			partner = list[0]
		}
	} else if !p.asyncRcv && !p.asyncSend && !p.closed && !p.doClose {
		partner = m.threads[p.t2].peek()
	} else {
		return
	}

	tEvent := m.threads[p.t1].peek().String()
	tEvent += "," + fmt.Sprintf("%v", p.idx) // + m.threads[p.t1].peek().ops[p.idx].String()
	partners := possiblePaths[tEvent]
	if partners == nil {
		partners = make(map[string]struct {
			covered int
			loc     string
		})
	}
	partners[partner.thread] = struct {
		covered int
		loc     string
	}{0, partner.String() + fmt.Sprintf(",%v", p.t2Idx)}
	possiblePaths[tEvent] = partners
}

/*
 todo:
  - if a rcv syncs with a send from a select, add the idx of the select case!
  - add select coverage
*/
func simulate(machines []machine) {
	possiblePaths := make(map[string]map[string]struct {
		covered int
		loc     string
	})
	receivesOnClosed := make(map[string]struct{})
	//	f, _ := os.Create("./bla.txt")
	//debugging
	//reader := bufio.NewReader(os.Stdin)
	//

	for {
		done := 0
		for _, m := range machines {
			if m.stopped {
				done++
			}
		}
		if done == len(machines) {
			addSendAlts(possiblePaths)
			for k, v := range possiblePaths {
				fmt.Printf("Partners for ")
				color.HiMagenta(k)
				for _, v2 := range v {
					if v2.covered == 0 {
						color.HiRed(v2.loc)
					} else {
						color.HiGreen(v2.loc)
					}
				}
			}
			fmt.Println("\nReceives on closed channels")
			for k := range receivesOnClosed {
				fmt.Println("\t", k)
			}

			return
		}

		for i := range machines {
			if machines[i].stopped {
				continue
			}

			pairs := machines[i].getSyncPairs()
			// f.WriteString(fmt.Sprintf("%v,%v,%v\n", i, len(machines), pairs))
			// for _, t := range machines[i].threads {
			// 	f.WriteString(fmt.Sprintf("(%v,%v)", t.ID, len(t.events)))
			// }
			// f.WriteString("\n")
			//	pairs = append(pairs, machines[i].getAsyncActions()...)
			// fmt.Println(i, pairs)
			// reader.ReadString('\n')
			if len(pairs) == 0 {
				machines[i].stopped = true
				continue
			}

			for j := range pairs {
				machines[i].updatePossiblePaths(pairs[j], possiblePaths)
				if pairs[j].asyncSend || pairs[j].asyncRcv {
					if j+1 < len(pairs) {
						//new machine
						nm := machines[i].clone()
						nm.asyncAction(pairs[j], possiblePaths)
						machines = append(machines, nm)
					} else {
						//'reuse' old machine
						machines[i].asyncAction(pairs[j], possiblePaths)
					}
				} else if pairs[j].closed {
					if j+1 < len(pairs) {
						//new machine
						nm := machines[i].clone()
						nm.closedRcv(pairs[j], receivesOnClosed)
						machines = append(machines, nm)
					} else {
						//'reuse' old machine
						machines[i].closedRcv(pairs[j], receivesOnClosed)
					}
				} else if pairs[j].doClose {
					if j+1 < len(pairs) {
						nm := machines[i].clone()
						nm.closeChan(pairs[j])
						machines = append(machines, nm)
					} else {
						machines[i].closeChan(pairs[j])
					}
				} else {
					if j+1 < len(pairs) {
						//new machine
						nm := machines[i].clone()
						nm.sync(pairs[j], possiblePaths)
						machines = append(machines, nm)
					} else {
						//'reuse' old machine
						machines[i].sync(pairs[j], possiblePaths)
					}
				}
			}
		}
	}
}

func getAsyncChans(items []Item) map[string][]Item {
	chans := make(map[string][]Item)
	for _, i := range items {
		for _, o := range i.ops {
			if o.bufSize == 0 {
				continue
			}
			_, ok := chans[o.ch]
			if !ok {
				chans[o.ch] = []Item{}
			}
		}
	}
	return chans
}

func addVCs(m *machine) {
	var items []Item
	chosenPartner := make(map[string]Item)
	for {
		pairs := m.getSyncPairs()
		pairs = append(pairs, m.getAsyncActions()...)
		if len(pairs) == 0 {
			break
		}

		for _, p := range pairs {
			if p.asyncSend {
				t1 := m.threads[p.t1]
				t1.vc.Add(t1.ID, 1)
				t1.events[0].vc = t1.vc.clone()
				t1.vc.Add(t1.ID, 1)
				t1.events[1].vc = t1.vc.clone()
				list := m.asyncChans[t1.events[1].ops[0].ch]
				list = append(list, t1.events[0])
				m.asyncChans[t1.events[0].ops[0].ch] = list
				items = append(items, t1.events[0])
				items = append(items, t1.events[1])
				t1.pop()
				t1.pop()
				m.threads[t1.ID] = t1
				continue
			} else if p.asyncRcv {
				list := m.asyncChans[p.t2]
				if len(list) > 0 {
					partner := list[0]
					t1 := m.threads[p.t1]

					if partner.thread != t1.events[1].partner {
						continue
					}

					if len(list) > 1 {
						list = list[1:]
					} else {
						list = []Item{}
					}
					m.asyncChans[p.t2] = list

					t1.vc.Add(t1.ID, 1)
					t1.events[0].vc = t1.vc.clone()
					t1.vc.Add(t1.ID, 1)
					t1.vc.sync(partner.vc)
					t1.events[1].vc = t1.vc.clone()
					items = append(items, t1.events[0])
					items = append(items, t1.events[1])

					chosenPartner[t1.events[0].ShortString()] = partner
					chosenPartner[partner.ShortString()] = t1.events[0]

					t1.pop()
					t1.pop()
					m.threads[t1.ID] = t1
				}
				continue
			}

			if p.doClose {
				t1 := m.threads[p.t1]
				t1.vc.Add(t1.ID, 1)
				t1.events[0].vc = t1.vc.clone()
				t1.vc.Add(t1.ID, 1)
				t1.events[1].vc = t1.vc.clone()
				m.closedChansVC[p.t2] = t1.vc.clone()
				m.closedChans[p.t2] = struct{}{}
				items = append(items, t1.events[0])
				items = append(items, t1.events[1])
				t1.pop()
				t1.pop()
				m.threads[t1.ID] = t1
				continue
			}

			if p.closed {
				t1 := m.threads[p.t1]
				t1.vc.Add(t1.ID, 1)
				t1.events[0].vc = t1.vc.clone()
				t1.vc.Add(t1.ID, 1)
				t1.events[1].vc = t1.vc.clone()
				items = append(items, t1.events[0])
				items = append(items, t1.events[1])
				t1.pop()
				t1.pop()
				m.threads[t1.ID] = t1
				continue
			}
			t1 := m.threads[p.t1]
			t2 := m.threads[p.t2]
			if len(t1.events) > 1 {
				if t1.events[1].partner != t2.ID {
					continue
				}

				//prepare
				t1.vc.Add(t1.ID, 1)
				t1.events[0].vc = t1.vc.clone()
				fmt.Println(t1.ID, t1.events[0], t2.ID, t2.vc)
				t2.vc.Add(t2.ID, 1)
				t2.events[0].vc = t2.vc.clone()

				//commit
				t1.vc.Add(t1.ID, 1)
				t2.vc.Add(t2.ID, 1)
				t1.vc.sync(t2.vc)
				t2.vc.sync(t1.vc)
				t1.events[1].vc = t1.vc.clone()
				t2.events[1].vc = t2.vc.clone()

				chosenPartner[t1.events[0].ShortString()] = t2.peek()
				chosenPartner[t2.events[0].ShortString()] = t1.peek()

				items = append(items, t1.events[0])
				items = append(items, t1.events[1])
				items = append(items, t2.events[0])
				items = append(items, t2.events[1])
				t1.pop()
				t1.pop()
				t2.pop()
				t2.pop()
				m.threads[t1.ID] = t1
				m.threads[t2.ID] = t2
			} else {
				fmt.Println(5)
				t1.vc.Add(t1.ID, 1)
				t1.events[0].vc = t1.vc.clone()
				items = append(items, t1.events[0])
				t1.pop()
				m.threads[p.t1] = t1
			}
		}
	}
	for _, t := range m.threads {
		for {
			if len(t.events) > 0 {
				t.vc.Add(t.ID, 1)
				t.events[0].vc = t.vc.clone()
				items = append(items, t.peek())
				t.pop()
				m.threads[t.ID] = t
			} else {
				break
			}
		}

	}
	// for _, it := range items {
	// 	fmt.Println(it)
	// }
	// fmt.Printf("\n\n")

	noAlts := true
	for _, it := range items {
		alternatives := make(map[string]struct{}) // []Item
		for _, op := range it.ops {
			opKind := OpKind(0)
			if op.kind == OpKind(PREPARE|RCV) {
				opKind = PREPARE | SEND
			} else if op.kind == PREPARE|SEND {
				opKind = PREPARE | RCV
			} else {
				continue
			}

			//var alternatives []Item
			for _, x := range items {
				for _, y := range x.ops {
					if y.kind != opKind /*OpKind(PREPARE|SEND)*/ || y.ch != op.ch || x.thread == it.partner || x.thread == it.thread {
						continue
					}
					if !x.vc.less(it.vc) && !it.vc.less(x.vc) {
						//	alternatives = append(alternatives, x)
						alternatives[x.ShortString()] = struct{}{}
					}
				}
			}
			// for k := range alternatives {
			// 	fmt.Println("\t", k)
			// }
		}
		if len(alternatives) > 0 {
			noAlts = false
			fmt.Println("Alternatives for", it)
			// for _, x := range alternatives {
			// 	color.HiRed(fmt.Sprintf("\t%v", x))
			// }
			vv, ok := chosenPartner[it.ShortString()]
			for k := range alternatives {
				if ok && vv.ShortString() == k {
					continue
				}
				color.HiRed(fmt.Sprintf("\t%v", k))
			}

			if ok {
				color.HiGreen(fmt.Sprintf("\t%v", vv.ShortString()))
			}

		}
	}
	if noAlts {
		fmt.Println("No alternatives found!")
	}
	// for _, it := range items {
	// 	fmt.Println(">>>", it)
	// }

	// for k, v := range chosenPartner {
	// 	fmt.Println(">>>", k, v)
	// }

	//check for close operations and what operations happend parallel on the same channel
	//this way we can find another kind of alternative schedules! in which for example a receive or even a send happened
	//shortly before the close but also could have happend afterwards!
	fmt.Printf("\n\n")
	for _, it := range items {
		for _, op := range it.ops {
			if op.kind != COMMIT|CLS {
				continue
			}
			//var altNAfter []Item
			altNAfter := make(map[string]int)
			for _, x := range items {
				for _, y := range x.ops {
					if y.ch != op.ch || y.kind&CLS > 0 {
						continue
					}
					v := 0
					if y.kind&SEND > 0 {
						v = 1
					}
					if !x.vc.less(it.vc) && !it.vc.less(x.vc) {
						altNAfter[x.ShortString()] = v
						//	altNAfter = append(altNAfter, x)
					} else if it.vc.less(x.vc) {
						altNAfter[x.ShortString()] = v
						//	altNAfter = append(altNAfter, x)
					}
				}
			}

			fmt.Println("Actions parallel or after", it)
			for k, v := range altNAfter {
				if v > 0 {
					color.HiRed(fmt.Sprintf("\t%v", k))
				} else {
					color.HiGreen(fmt.Sprintf("\t%v", k))
				}
			}
		}
	}
}

func main() {
	color.HiGreen("Covered schedules")
	color.HiRed("Uncovered schedules")
	color.HiYellow("-----------------------")
	trace := flag.String("trace", "", "path to trace")
	flag.Parse()

	if trace == nil || *trace == "" {
		panic("no valid trace file")
	}

	items := parseTrace(*trace)
	threads := createThreads(items)
	aChans := getAsyncChans(items)

	// for _, x := range items {
	// 	fmt.Println(x)
	// }
	// fmt.Println("--------")
	// for _, t := range threads {
	// 	fmt.Println(t.ID)
	// 	for _, v := range t.events {
	// 		fmt.Println("\t", v)
	// 	}
	// }
	closed := make(map[string]struct{})
	closed["0"] = struct{}{}
	// simulate([]machine{machine{threads, aChans, closed, false}})
	addVCs(&machine{threads, aChans, closed, make(map[string]VectorClock), false})

}