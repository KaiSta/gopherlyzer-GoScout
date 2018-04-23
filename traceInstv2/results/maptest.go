package main

import "fmt"

func main() {
	tracer.RegisterThread("main")
	m := make(map[int]int)
	tracer.WriteAcc(&m, "tests\\maptest.go:6", tracer.GetGID())
	c := make(chan int)
	tracer.RegisterChan(c, cap(c))

	x := m[0]
	tracer.WriteAcc(&x, "tests\\maptest.go:9", tracer.GetGID())
	tracer.ReadAcc(&m, "tests\\maptest.go:9", tracer.GetGID())
	tracer.WriteAcc(&x, "tests\\maptest.go:10", tracer.GetGID())
	x++
	m[0] = x
	tracer.WriteAcc(&m, "tests\\maptest.go:11", tracer.GetGID())
	tracer.ReadAcc(&x, "tests\\maptest.go:11", tracer.GetGID())
	tracer.ReadAcc(&m, "tests\\maptest.go:13", tracer.GetGID())
	fmt.Println(m[0])

	z := make([]int, 2)
	tracer.WriteAcc(&z, "tests\\maptest.go:15", tracer.GetGID())
	z[1] = 1
	tracer.WriteAcc(&z[1], "tests\\maptest.go:16", tracer.GetGID())
	z[0] = 0
	tracer.WriteAcc(&z[0], "tests\\maptest.go:17", tracer.GetGID())
	tracer.ReadAcc(&z[0], "tests\\maptest.go:18", tracer.GetGID())
	fmt.Println(z[0])
	tracer.ReadAcc(&z, "tests\\maptest.go:19", tracer.GetGID())
	fmt.Println(z)
}
