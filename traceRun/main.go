package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	// +=========+
	// |  Flags  |
	// +=========+
	// - Variables
	var instFilePath string
	flag.StringVar(&instFilePath, "instFilePath", "", "Specify path to instrumented Go file")
	var logPath string
	flag.StringVar(&logPath, "logPath", "", "Specifies output filename and path of the log file")

	flag.Parse()

	// +--------------+
	// |  Code start  |
	// +--------------+
	// Checks for need to run
	if instFilePath == "" {
		fmt.Println("instFilePath not specified\n")
		flag.Usage()
		return
	}

	// Runs edited Go file
	cmd := exec.Command("go", "run", instFilePath)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}

	// if out.Len() > 0 {
	// 	fmt.Println("Output: " + out.String())
	// }

	if logPath != "" {
		fmt.Println("Copying trace.log to", logPath)
		err := Copy("trace.log", logPath)
		if err != nil {
			panic(err)
		}
	}
}

// Copy - Copies a file from src to dst
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
