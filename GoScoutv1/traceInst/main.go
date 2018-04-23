package main

import (
	"flag"
	"fmt"

	"./engine"
)

func main() {
	in := flag.String("in", "", "path to code file")
	out := flag.String("out", "", "result path")
	flag.Parse()
	fmt.Println(*in, "|", *out)
	if in == nil || *in == "" {
		panic("no valid in path")
	}
	if out == nil || *out == "" {
		panic("no valid out path")
	}

	p := engine.NewASTParser(*in, *out)
	p.Run()
}
