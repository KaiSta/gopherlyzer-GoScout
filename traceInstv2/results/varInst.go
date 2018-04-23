package main

import "fmt"

var g int

func main() {

	var y string
	tracer.WriteAcc(z, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\varInst.go:28")
	z := 0
	tracer.WriteAcc(g, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\varInst.go:30")
	g++
	tracer.ReadAcc(y, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\varInst.go:31")
	tracer.ReadAcc(z, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\varInst.go:31")
	tracer.ReadAcc(g, "C:\\Users\\stka0001\\Github\\gopherlyzer-GoScout\\traceInst\\tests\\varInst.go:31")
	fmt.Println(y, z, g)
}
