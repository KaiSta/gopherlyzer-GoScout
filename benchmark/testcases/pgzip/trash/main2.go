package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"../../../../SpeedyGo/traceInstv2/tracer"
	"../../../pgzip3n"
)

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	buf := new(bytes.Buffer)

	w := pgzip3n.NewWriter(buf)
	w.Comment = "comment"
	w.Extra = []byte("extra")
	w.ModTime = time.Unix(1e8, 0)
	w.Name = "name"
	if _, err := w.Write([]byte("payload")); err != nil {
		fmt.Println("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		fmt.Println("Writer.Close: %v", err)
	}

	r, err := pgzip3n.NewReader(buf)
	if err != nil {
		fmt.Println("NewReader: %v", err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		fmt.Println("ReadAll: %v", err)
	}
	if string(b) != "payload" {
		fmt.Println("payload is %q, want %q", string(b), "payload")
	}
	if r.Comment != "comment" {
		fmt.Println("comment is %q, want %q", r.Comment, "comment")
	}
	if string(r.Extra) != "extra" {
		fmt.Println("extra is %q, want %q", r.Extra, "extra")
	}
	if r.ModTime.Unix() != 1e8 {
		fmt.Println("mtime is %d, want %d", r.ModTime.Unix(), uint32(1e8))
	}
	if r.Name != "name" {
		fmt.Println("name is %q, want %q", r.Name, "name")
	}
	if err := r.Close(); err != nil {
		fmt.Println("Reader.Close: %v", err)
	}
	tracer.Stop()
}
