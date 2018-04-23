package util

import "fmt"

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

	if k&WAIT > 0 {
		sym = "W"
		stat = "C"
	} else if k&SIG > 0 {
		sym = "S"
		stat = "C"
	}

	if k&WRITE > 0 {
		sym = "M"
		stat = "C"
	} else if k&READ > 0 {
		sym = "R"
		stat = "C"
	}
	return fmt.Sprintf("%v,%v", sym, stat)
}

type Operation struct {
	Ch        string
	Kind      OpKind
	BufSize   int
	SourceRef string
	Mutex     int
}

func (o Operation) String() string {
	return fmt.Sprintf("%v(%v),%v,%v", o.Ch, o.BufSize, o.Kind, o.SourceRef)
}

type Item struct {
	Thread  string
	Ops     []Operation
	Partner string
	VC      VectorClock
}

func (o Item) Clone() *Item {
	var ops []Operation
	for _, x := range o.Ops {
		ops = append(ops, x)
	}
	vc := NewVC()
	for k, v := range o.VC {
		vc[k] = v
	}
	return &Item{o.Thread, ops, o.Partner, vc}
}

func (o Item) String() string {
	var ops string
	for i, p := range o.Ops {
		ops += fmt.Sprintf("(%v)", p)

		if i+1 < len(o.Ops) {
			ops += ","
		}
	}

	if o.Partner != "" {
		return fmt.Sprintf("%v,[%v],%v", o.Thread, ops, o.Partner)
	}
	return fmt.Sprintf("%v,[%v]", o.Thread, ops)
}
func (o Item) ShortString() string {
	var ops string
	for i, p := range o.Ops {
		ops += fmt.Sprintf("(%v)", p)

		if i+1 < len(o.Ops) {
			ops += ","
		}
	}

	if o.Partner != "" {
		return fmt.Sprintf("%v,[%v],%v", o.Thread, ops, o.Partner)
	}
	return fmt.Sprintf("%v,[%v]", o.Thread, ops)
}

type Thread struct {
	ID       string
	Events   []*Item
	VC       VectorClock
	RVC      VectorClock
	MutexSet map[string]struct{}
}

func (t Thread) String() string {
	return fmt.Sprintf("(%v, %v)", t.ID, t.Events)
}
func (t Thread) ShortString() string {
	return fmt.Sprintf("(%v, %v)", t.ID, t.Events[0])
}

func (t Thread) Clone() Thread {
	var items []*Item
	for i := range t.Events {
		items = append(items, t.Events[i])
	}
	vc := NewVC()
	for k, v := range t.VC {
		vc[k] = v
	}
	var mset map[string]struct{}
	for k, v := range t.MutexSet {
		mset[k] = v
	}
	rvc := NewVC()
	for k, v := range t.RVC {
		rvc[k] = v
	}
	return Thread{t.ID, items, vc, rvc, mset}
	//	return Thread{t.ID, t.isBlocked, t.done, items, t.systemState}
}

func (t Thread) Peek() *Item {
	return t.Events[0]
}
func (t *Thread) Pop() {
	if len(t.Events) > 1 {
		t.Events = t.Events[1:]
	} else {
		t.Events = []*Item{}
	}
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
func (vc VectorClock) Sync(pvc VectorClock) {
	for k, v := range vc {
		pv := pvc[k]
		tmp := Max(v, pv)
		//vc[k] = Max(v, pv)
		vc[k] = tmp
		pvc[k] = tmp
	}
	for k, v := range pvc {
		pv := vc[k]
		tmp := Max(v, pv)
		vc[k] = tmp
		pvc[k] = tmp
	}
}

func (vc VectorClock) Equals(pvc VectorClock) bool {
	if len(vc) != len(pvc) {
		return false
	}
	for k, v := range vc {
		pv := pvc[k]
		if v != pv {
			return false
		}
	}
	return true
}

func (vc VectorClock) Remove(pvc VectorClock) VectorClock {
	nClock := NewVC()
	for k, v := range vc {
		w := pvc[k]
		if v > w {
			nClock[k] = v
		}
	}
	for k, w := range pvc {
		_, ok := vc[k]
		if !ok {
			vc[k] = w
		}
	}

	return nClock
}

func (vc VectorClock) AddEpoch(ep Epoch) {
	vc[ep.X] = ep.T
}

func (vc VectorClock) Add(k string, val int) {
	v := vc[k]
	v += val
	vc[k] = v
}
func (vc VectorClock) Set(k string, val int) {
	v := vc[k]
	v = val
	vc[k] = v
}
func (vc VectorClock) Less(pvc VectorClock) bool {
	if len(vc) == 0 { //??? not sure if that is ok
		return true
	}
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

func (vc VectorClock) FindConflict(pvc VectorClock) string {
	for k := range vc {
		if vc[k] > pvc[k] {
			return k
		}
	}
	for k := range pvc {
		if vc[k] > pvc[k] {
			return k
		}
	}
	return ""
}

func (vc VectorClock) ConcurrentTo(pvc VectorClock) bool {
	if vc.Less(pvc) || pvc.Less(vc) {
		return false
	}
	return true
}

func (vc VectorClock) Less_Epoch(pepoch Epoch) bool {
	return vc[pepoch.X] < pepoch.T
}

func (vc VectorClock) Clone() VectorClock {
	nvc := NewVC()
	for k, v := range vc {
		nvc[k] = v
	}
	return nvc
}

func NewVC() VectorClock {
	return make(VectorClock)
}

type Result struct {
	Alts []Alternative2
	POs  []Alternative2
}
type Alternative struct {
	Op     string   `json:"op"`
	Used   []string `json:"used"`
	Unused []string `json:"unused"`
}
type Alternative2 struct {
	Op     *Item
	Used   []*Item
	Unused []*Item
}

type VarState1 struct {
	Rvc     VectorClock
	Wvc     VectorClock
	PrevREv *Item
	PrevWEv *Item
}

type Epoch struct {
	X string
	T int
}

func NewEpoch(x string, t int) Epoch {
	return Epoch{x, t}
}
func (e *Epoch) Set(x string, t int) {
	e.X = x
	e.T = t
}
func (e Epoch) Less_Epoch(vc VectorClock) bool {
	return e.T < vc[e.X]
}

type VarState2 struct {
	Rvc     VectorClock
	Wepoch  Epoch
	Repoch  Epoch
	PrevREv *Item
	PrevWEv *Item
}

type VState int

func (s VState) String() string {
	switch s {
	case EXCLUSIVE:
		return "EXCLUSIVE"
	case READSHARED:
		return "READSHARED"
	case SHARED:
		return "SHARED"
	}
	return ""
}

//VarState3 for Epoch + Eraser solution
type VarState3 struct {
	Rvc        VectorClock
	Wvc        VectorClock
	Wepoch     Epoch
	Repoch     Epoch
	State      VState
	LastAccess string
	LastOp     *Operation
	MutexSet   map[string]struct{}
	PrevREv    *Item
	PrevWEv    *Item
	TSet       VectorClock
}

type ChanState struct {
	Rvc      VectorClock
	Wvc      VectorClock
	WContext map[string]string
	RContext map[string]string
	//For RWLocks
	CountR int
}

func NewChanState() *ChanState {
	//return &ChanState{Rvc: NewVC(), Wvc: NewVC(), WContext: make([]string, 0), RContext: make([]string, 0)}
	return &ChanState{Rvc: NewVC(), Wvc: NewVC(), WContext: make(map[string]string), RContext: make(map[string]string)}
}
