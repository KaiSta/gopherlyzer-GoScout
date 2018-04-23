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
	t.x++
}

func (t someT) bar() {
	t.v--
}

func main() {
	t := test{z: &someT{42}}

	t.x++
	fmt.Println(t.y)
	t.z.v++
	t.a.v++

	t.foo(t.z.v)
	t.z.bar()
	t.a.bar()

	t.z.x++
	t.a.x--

	u := &otherT{42}

	m := false
	n := "hello"

	if u == nil {
		zz := nil
	}
}
