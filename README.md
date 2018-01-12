# gopherlyzer-GoScout

## What is GoScout?
GoScout is a suite of tools written in Golang with the purpose of creating value for Go developers, by identifying potential alternative program flows in message-passing Go programs.

What GoScout does:

- Analyzes message-passing programs by observing their run-time behavior through a purely library-based instrumentation method to trace communication events during execution
- Traces events that were unable to take place (This provides the user with additional information to identify potential bugs)
- Outputs the trace analysis results to the user, displaying the program flows that were covered and uncovered during that particular run

We have fully implemented our approach in the Go programming language and provide a number of examples to substantiate our claims. Compared to the vector clock method, the approach used in GoScout is much simpler and has in general, a significantly lower run-time overhead. 

All of Go's message-passing features are supported (details for how buffered channels, select with default/timeout as well as closing channels can be found in the appendix). However, our current prototype only supports instrumentation of a single Go file at a time and reports the set of covered/uncovered program flows on the command line.

Original research paper: [Trace-Based Run-time Analysis of Message-Passing Go Programs](https://www.home.hs-karlsruhe.de/~suma0002/publications/go-trace-based-run-time-analysis.pdf)



## Who is GoScout for?

Go developers who implement concurrency in their programs and can benefit from knowing how their programs could potentially be broken due to an erroneous order of events caused by alternative program flows.



## Usage

### *Instrumentation*

```bash
cd traceInst
go run main.go -in ../Tests/newsReader.go -out ../Tests/newsReaderInst.go
```

1. Adds instrumentation to newsReader.go and generates an output code file newsReaderInst.go

**Input**: Go file (newsReader.go)  
**Output**: Instrumented Go file (newsReaderInst.go)



### *Preparation*

```bash
cd tracePrep
go run main.go -instFilePath ../Tests/newsReaderInst.go -printFuncs
go run main.go -instFilePath ../Tests/newsReaderInst.go -function main -tracerPath "../traceInst/tracer" -overwrite
```

1. Prints list of functions in newsReaderInst.go
2. Adds codes to trace the "main" method and imports the **relative** path to the "tracer" package into newsReaderInst.go

**Input**: Instrumented Go file (newsReaderInst.go)  
**Output**: Prepared Go file (newsReaderInst.go, overwritten)



### *Running*

```bash
cd traceRun
go run main.go -instFilePath ../Tests/newsReader.go -logPath path/to/logs/newsReaderTrace.log
```

1. Runs newsReaderInst.go and saves the generated log file to the specified path (path/to/logs/newsReaderTrace.log)

**Input**: Prepared Go file (newsReaderInst.go)  
**Output**: Trace log (newsReaderTrace.log)



### *Verification*

```bash
cd traceVerify
go run main.go -plain -trace path/to/logs/newsReaderTrace.log
go run main.go -json -trace path/to/logs/newsReaderTrace.log
```

1. **Plain** option: Interprets newsReaderTrace.log and displays covered/uncovered program flows in a color-coded text format (viewable only on command line)
2. **JSON** option: Interprets newsReaderTrace.log and displays covered/uncovered programs in JSON format (to be parsed)

**Input**: Trace log (newsReaderTrace.log)  
**Output**: Covered/uncovered program flows (Plain/JSON)

```diff
Alternatives for fun14,[(2(0),?,P,go-examples\newsReader.go:25)]
+        bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)]
Alternatives for bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)]
-        fun15,[(2(0),?,P,go-examples\newsReader.go:25)]
+        fun14,[(2(0),?,P,go-examples\newsReader.go:25)]
Alternatives for fun03,[(1(0),?,P,go-examples\newsReader.go:21)]
+        reuters20,[(1(0),!,P,go-examples\newsReader.go:7)]
Alternatives for reuters20,[(1(0),!,P,go-examples\newsReader.go:7)]
-        fun06,[(1(0),?,P,go-examples\newsReader.go:21)]
+        fun03,[(1(0),?,P,go-examples\newsReader.go:21)]
Alternatives for fun15,[(2(0),?,P,go-examples\newsReader.go:25)]
-        bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)]
Alternatives for fun06,[(1(0),?,P,go-examples\newsReader.go:21)]
-        reuters20,[(1(0),!,P,go-examples\newsReader.go:7)]
Alternatives for newsReader41,[(4(0),?,P,go-examples\newsReader.go:28)]
+        fun03,[(4(0),!,P,go-examples\newsReader.go:21)]
Alternatives for fun03,[(4(0),!,P,go-examples\newsReader.go:21)]
+        newsReader41,[(4(0),?,P,go-examples\newsReader.go:28)]
Alternatives for main,[(3(0),?,P,go-examples\newsReader.go:28)]
+        fun14,[(3(0),!,P,go-examples\newsReader.go:25)]
Alternatives for fun14,[(3(0),!,P,go-examples\newsReader.go:25)]
+        main,[(3(0),?,P,go-examples\newsReader.go:28)]
```



## Credits

**Prof. Martin Sulzmann**  
Professor, Faculty of Computer Science and Business Information Systems  
Karlsruhe University of Applied Sciences (Germany)

**Kai Stadtmueller**  
MSc, Faculty of Computer Science and Business Information Systems  
Karlsruhe University of Applied Sciences (Germany)

**Nicholas Yap**  
Diploma in Cyber Security and Forensics  
Nanyang Polytechnic (Singapore)
