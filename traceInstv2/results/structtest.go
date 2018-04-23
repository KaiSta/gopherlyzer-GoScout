package main

import "fmt"

type test struct {
	x int
	y string
	z *someT
	a someT
}

type someT struct {
	otherT
	v int
}

type otherT struct {
	x float64
}

func (t *test) foo(v int) {
	tracer.WriteAcc(&t.x, "tests\\structtest.go:22", tracer.GetGID())
	t.x++
}

func (t someT) bar() {
	tracer.WriteAcc(&t.v, "tests\\structtest.go:26", tracer.GetGID())
	t.v--
}

func main() {
	tracer.RegisterThread("main")
	t := test{z: &someT{42}}
	tracer.WriteAcc(&t, "tests\\structtest.go:30", tracer.GetGID())
	tracer.WriteAcc(&t.x, "tests\\structtest.go:32", tracer.GetGID())
	t.x++
	tracer.ReadAcc(&t.y, "tests\\structtest.go:33", tracer.GetGID())
	fmt.Println(t.y)
	tracer.WriteAcc(&t.z.v, "tests\\structtest.go:34", tracer.GetGID())
	t.z.v++
	tracer.WriteAcc(&t.a.v, "tests\\structtest.go:35", tracer.GetGID())
	t.a.v++
	tracer.ReadAcc(&t.z.v, "tests\\structtest.go:37", tracer.GetGID())
	t.foo(t.z.v)
	t.z.bar()
	t.a.bar()
	tracer.WriteAcc(&t.z.x, "tests\\structtest.go:41", tracer.GetGID())
	t.z.x++
	tracer.WriteAcc(&t.a.x, "tests\\structtest.go:42", tracer.GetGID())
	t.a.x--

	u := &otherT{42}
	tracer.WriteAcc(&u, "tests\\structtest.go:44", tracer.GetGID())

	m := false
	tracer.WriteAcc(&m, "tests\\structtest.go:46", tracer.GetGID())
	n := "hello"
	tracer.WriteAcc(&n, "tests\\structtest.go:47", tracer.GetGID())
	tracer.ReadAcc(&u, "tests\\structtest.go:49", tracer.GetGID())
	if u == nil {
		zz := nil
		tracer.WriteAcc(&zz, "tests\\structtest.go:50", tracer.GetGID())
	}
}
