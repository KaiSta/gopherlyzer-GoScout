package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"os"

	"../../pgzip"
	"../../tracer"
)

func main() {
	tracer.Start()
	tracer.RegisterThread("main")
	buf := new(bytes.Buffer)

	f, _ := os.Open("fb18.pdf")
	x, _ := ioutil.ReadAll(f)

	w := pgzip.NewWriter(buf)
	w.Comment = "comment"
	w.Extra = []byte("extra")
	w.ModTime = time.Unix(1e8, 0)
	w.Name = "name"
	if _, err := w.Write(x); err != nil {
		fmt.Printf("Write: %v\n", err)
	}
	if err := w.Close(); err != nil {
		fmt.Printf("Writer.Close: %v\n", err)
	}

	// ioutil.WriteFile("result.gzip", buf.Bytes(), 0644)

	// f2, _ := os.Open("result.gzip")
	// r, err := pgzip2.NewReader(f2)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// b, err := ioutil.ReadAll(r)
	// if err != nil {
	// 	fmt.Println(err)
	// 	//return
	// }
	// fmt.Println(len(b), len(x), len(x)-len(b))

	// buf2 := new(bytes.Buffer)
	// w2 := pgzip2.NewWriter(buf2)
	// w2.Comment = "comment"
	// w2.Extra = []byte("extra")
	// w2.ModTime = time.Unix(1e8, 0)
	// w2.Name = "name"
	// if _, err := w2.Write([]byte("payload")); err != nil {
	// 	fmt.Printf("Write: %v\n", err)
	// }
	// if err := w2.Close(); err != nil {
	// 	fmt.Printf("Writer.Close: %v\n", err)
	// }
	// fmt.Println("------")
	// fmt.Println(buf)
	// fmt.Println("------")
	// fmt.Println(buf2)
	// fmt.Println("------")
	// b1 := buf.Bytes()
	// b2 := buf2.Bytes()
	// if len(b1) != len(b2) {
	// 	fmt.Println("FUU", len(b1), len(b2))
	// }

	// r, err := pgzip2.NewReader(buf)
	// if err != nil {
	// 	fmt.Printf("1NewReader: %v\n", err)
	// }
	// b, err := ioutil.ReadAll(r)
	// if err != nil {
	// 	fmt.Printf("1ReadAll: %v\n", err)
	// }
	// if string(b) != "payload" {
	// 	fmt.Printf("1payload is %q, want %q\n", string(b), "payload")
	// }
	// if r.Comment != "comment" {
	// 	fmt.Printf("1comment is %q\n, want %q\n", r.Comment, "comment")
	// }
	// if string(r.Extra) != "extra" {
	// 	fmt.Printf("1extra is %q\n, want %q\n", r.Extra, "extra")
	// }
	// if r.ModTime.Unix() != 1e8 {
	// 	fmt.Printf("1mtime is %d, want %d", r.ModTime.Unix(), uint32(1e8))
	// }
	// if r.Name != "name" {
	// 	fmt.Printf("1name is %q\n, want %q\n", r.Name, "name")
	// }
	// if err := r.Close(); err != nil {
	// 	fmt.Printf("1Reader.Close: %v\n", err)
	// }

	// r2, err := pgzip.NewReader(buf2)
	// if err != nil {
	// 	fmt.Printf("2NewReader: %v\n", err)
	// }
	// b, err = ioutil.ReadAll(r2)
	// if err != nil {
	// 	fmt.Printf("2ReadAll: %v\n", err)
	// }
	// if string(b) != "payload" {
	// 	fmt.Printf("2payload is %q, want %q\n", string(b), "payload")
	// }
	// if r.Comment != "comment" {
	// 	fmt.Printf("2comment is %q\n, want %q\n", r2.Comment, "comment")
	// }
	// if string(r.Extra) != "extra" {
	// 	fmt.Printf("2extra is %q\n, want %q\n", r2.Extra, "extra")
	// }
	// if r.ModTime.Unix() != 1e8 {
	// 	fmt.Printf("2mtime is %d, want %d", r2.ModTime.Unix(), uint32(1e8))
	// }
	// if r.Name != "name" {
	// 	fmt.Printf("2name is %q\n, want %q\n", r2.Name, "name")
	// }
	// if err := r2.Close(); err != nil {
	// 	fmt.Printf("2Reader.Close: %v\n", err)
	// }
	tracer.Stop()
}
