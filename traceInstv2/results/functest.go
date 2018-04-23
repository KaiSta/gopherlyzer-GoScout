package main

import (
	"fmt"
	"sync"
)

func foo(x *sync.Mutex) {

}

func bar(x, y int) (int, int) {
	tracer.WriteAcc(&x, "tests\\functest.go:13", tracer.GetGID())
	x++
	tracer.WriteAcc(&y, "tests\\functest.go:14", tracer.GetGID())
	y++
	tracer.ReadAcc(&x, "tests\\functest.go:15", tracer.GetGID())
	fmt.Println(x)
	return x, y
}

func baz() int {
	return 42
}

func main() {
	tracer.RegisterThread("main")
	a := sync.Mutex{}
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, tracer.GetGID())

	go func(x *sync.Mutex) { tracer.RegisterThread("foo0"); tracer.Wait(tmp1, tracer.GetGID()); foo(x) }(&a)
	x := baz()
	tracer.WriteAcc(&x, "tests\\functest.go:27", tracer.GetGID())
	tracer.ReadAcc(&x, "tests\\functest.go:28", tracer.GetGID())
	v1, v2 := bar(5, x)
	tracer.WriteAcc(&v1, "tests\\functest.go:28", tracer.GetGID())
	tracer.WriteAcc(&v2, "tests\\functest.go:28", tracer.GetGID())
	tracer.ReadAcc(&v1, "tests\\functest.go:30", tracer.GetGID())
	tracer.ReadAcc(&v2, "tests\\functest.go:30", tracer.GetGID())
	fmt.Println(v1, v2)

	v1, _ = bar(4, 5)
	tracer.WriteAcc(&v1, "tests\\functest.go:32", tracer.GetGID())

	y := make([]int, 2)
	tracer.WriteAcc(&y, "tests\\functest.go:34", tracer.GetGID())
	v3 := y[0]
	tracer.WriteAcc(&v3, "tests\\functest.go:35", tracer.GetGID())
	tracer.ReadAcc(&y[0], "tests\\functest.go:35", tracer.GetGID())
}
