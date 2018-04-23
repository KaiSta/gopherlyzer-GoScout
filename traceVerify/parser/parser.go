package parser

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"../util"
)

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

func getOps(l string) ([]util.Operation, string) {
	var ops []util.Operation
	s := 0
	i := 0
	var curr util.Operation
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
				curr = util.Operation{}
				s++
			} else {
				panic("invalid trace format")
			}
		case 2:
			if c == ',' {
				s++
			} else {
				curr.Ch += string(c)
			}
		case 3:
			if c == ',' {
				bufsize, _ := strconv.ParseUint(currBufSize, 10, 64)
				curr.BufSize = int(bufsize)
				currBufSize = ""
				s++
			} else {
				currBufSize += string(c)
			}
		case 4:
			if c == '!' {
				curr.Kind |= util.SEND
			} else if c == '?' {
				curr.Kind |= util.RCV
			} else if c == 'C' {
				curr.Kind |= util.CLS
			} else if c == 'W' {
				curr.Kind |= util.WAIT
			} else if c == 'S' {
				curr.Kind |= util.SIG
			} else if c == 'M' {
				curr.Kind |= util.WRITE
			} else if c == 'R' {
				curr.Kind |= util.READ
			} else if c == '+' {
				curr.Kind |= util.SEND
				curr.Mutex |= util.LOCK
			} else if c == '$' {
				curr.Kind |= util.SEND
				curr.Mutex |= util.RLOCK
			} else if c == '#' {
				curr.Kind |= util.RCV
				curr.Mutex |= util.RUNLOCK
			} else if c == '*' {
				curr.Kind |= util.RCV
				curr.Mutex |= util.UNLOCK
			} else if c == ',' {
				s++
			} else {
				panic(fmt.Errorf("Unkown op %v .", string(c)))
			}
		case 5:
			if c == ')' {
				ops = append(ops, curr)
				s++
			} else {
				curr.SourceRef += string(c)
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
			ops[i].Kind |= util.COMMIT
		} else {
			ops[i].Kind |= util.PREPARE
		}
	}
	i++

	return ops, l[i:]
}

func ParseTrace(s string) []util.Item {
	data, err := ioutil.ReadFile(s)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(data), "\n")

	var items []util.Item
	for _, l := range lines {
		if len(l) == 0 || len(l) == 1 {
			break
		}
		var item util.Item
		var rest string
		item.Thread, rest = getTName(l)
		item.Ops, rest = getOps(rest)
		item.Partner, rest = getTName(rest)

		items = append(items, item)
	}

	return items
}
