package report

import (
	"fmt"
	"strings"

	"../../util"
	"github.com/fatih/color"
)

const (
	SEVERE = iota << 1
	NORMAL = iota << 1
	LOW    = iota << 1
)

type level int

var messageCache map[string]struct{}

func init() {
	messageCache = make(map[string]struct{})
}

func Race(op1, op2 *util.Operation, importance level) {
	s := fmt.Sprintf("Race between:\n1.%v\n2.%v\n\n", op1.SourceRef, op2.SourceRef)

	_, ok := messageCache[s]

	if ok {
		return
	}

	s2 := fmt.Sprintf("Race between:\n1.%v\n2.%v\n\n", op2.SourceRef, op1.SourceRef)

	messageCache[s] = struct{}{}
	messageCache[s2] = struct{}{}

	switch importance {
	case SEVERE:
		color.HiRed("Race between:\n1.%v\n2.%v\n\n", op1.SourceRef, op2.SourceRef)
	case NORMAL:
		color.HiBlue("Race between:\n1.%v\n2.%v\n\n", op1.SourceRef, op2.SourceRef)
	case LOW:
		color.HiGreen("Race between:\n1.%v\n2.%v\n\n", op1.SourceRef, op2.SourceRef)
	}
}

var alts map[string]map[string]int
var altCount int
var selectAltCount int

func Alternative(op, alt string) {
	if alts == nil {
		alts = make(map[string]map[string]int)
	}

	line := alts[op]
	if line == nil {
		line = make(map[string]int)
	}
	x, ok := line[alt]
	if !ok {
		line[alt] = 1 //new alternative
	} else {
		line[alt] = x + 1
	}
	alts[op] = line
	altCount++
}

func ReportAlts() {

	for k, v := range alts {
		allAcc := 0
		ownAcc := 0
		fmt.Printf("Alternatives for %v:\n", k)
		for x, y := range v {
			fmt.Printf("\t%v, %v\n", x, y)
			allAcc += y
			tmp := strings.Split(k, ":")
			if strings.Contains(x, tmp[0]) {
				ownAcc += y
			}
		}
		fmt.Printf("Ratio:%v/%v\n", ownAcc, allAcc)
		fmt.Printf("\n\n")
	}

	for k, v := range selAlts {
		fmt.Printf("Alternatives for select %v:\n", k)
		for x, y := range v {
			fmt.Printf("\t%v, %v\n", x, y)
		}
		fmt.Printf("\n\n")
	}
}

var selAlts map[string]map[string]int

func SelectAlternative(sel, op string) {
	if selAlts == nil {
		selAlts = make(map[string]map[string]int)
	}

	line := selAlts[sel]
	if line == nil {
		line = make(map[string]int)
	}
	x, ok := line[op]
	if !ok {
		line[op] = 1 //new alternative
	} else {
		line[op] = x + 1
	}
	selAlts[sel] = line
	selectAltCount++
}

type eventPair struct {
	pre  *util.Item
	post *util.Item
}

var EventsList map[string][]eventPair

func AddEvent(pre, post *util.Item) {
	if EventsList == nil {
		EventsList = make(map[string][]eventPair)
	}
	line := EventsList[pre.Thread]
	line = append(line, eventPair{pre, post})
	EventsList[pre.Thread] = line
}

func Events() {
	for k, v := range EventsList {
		fmt.Println(k)
		for _, x := range v {
			fmt.Printf("\t%v,%v\n", x.pre, x.pre.VC)
		}
	}
}

func AlternativesFromReport() {
	count := 0
	for k, v := range EventsList {
		fmt.Println("Alternatives for thread", k)
		for _, a := range v {
			alts := alternativesForEvent(a)
			count += len(alts)
			if len(a.pre.Ops) > 1 {
				selectAltCount++
			}
			idx := findChosenAlt(a, alts)
			if len(alts) > 1 {
				fmt.Printf("\tFor Event %v\n", a.pre)
				for i, b := range alts {
					fmt.Printf("\t\t%v-%v\n", i == idx, b.pre)
				}
			}
		}
	}
	fmt.Println("!!!!!!", "reportalts:", count, "on-fly alts:", altCount, "selalts", selectAltCount)
}

func findChosenAlt(it eventPair, alts []eventPair) int {
	if it.post == nil {
		return -1
	}
	//sync case
	for i, a := range alts {
		if a.post != nil && it.post.VC.Equals(a.post.VC) {
			return i
		}
	}
	//async case
	if it.pre.Ops[0].Kind&util.RCV == 0 {
		return 0
	}
	for i, a := range alts {
		if a.post == nil {
			continue
		}
		itVC := it.pre.VC.Clone()
		aVC := a.post.VC.Clone()
		itVC.Add(it.pre.Thread, 1)
		itVC.Sync(aVC)

		if itVC.Equals(it.post.VC) {
			return i
		}
	}
	return -1
}

func alternativesForEvent(it eventPair) []eventPair {
	alts := []eventPair{}
	for k, v := range EventsList {
		if k == it.pre.Thread {
			continue
		}
		for _, a := range v {
			if !match(a.pre, it.pre) {
				continue
			}

			if a.pre.VC.ConcurrentTo(it.pre.VC) {
				alts = append(alts, a)
			}
			if it.pre.VC.Less(a.pre.VC) {
				break
			}
		}
	}
	return alts
}

func match(it1, it2 *util.Item) bool {
	for _, op1 := range it1.Ops {
		for _, op2 := range it2.Ops {
			if singleMatch(op1, op2) {
				return true
			}
		}
	}
	return false
}

func singleMatch(op1, op2 util.Operation) bool {
	if op1.Ch != op2.Ch {
		return false
	}
	if op1.Kind == op2.Kind {
		return false
	}
	return true
}
