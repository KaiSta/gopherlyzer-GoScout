# gopherlyzer-GoScout

Prototype implementation for our analysis described in 'Trace-Based Run-time Analysis Method of Message-Passing Go Programs'. 
It still contains bugs and covers only a subset of the Go language. See our examples for what it is able to do at the moment.

## Description

We consider the task of analyzing message-passing programs
by observing their run-time behavior.
We introduce a simple instrumentation method to trace communication events
during execution. A model of the dependencies among events can
be constructed to identify  potential bugs.
Compared to the vector clock method, our approach is much simpler and
has in general a significant lower run-time overhead.
A further advantage is that we also trace events
that could not commit. Thus, we can infer
alternative communications which provides useful information to the user.
We have fully implemented our approach in the Go programming language
and provide a number of examples to substantiate our claims.

## How to use

Example:

### Instrumentation

cd traceInst

go run main -in ../Tests/newsReader.go -out ../Tests/newsReaderInst.go

Open newsReaderInst.go and add 

import "../traceInst/tracer" 

after the package declaration.

Add tracer.Start() at the beginning of the main function and tracer.Stop at the end.

### Run
Running the instrumented code with

go run newsReaderInst.go

will produce a trace.log file in the same folder.

### Verification

cd traceVerify

go run main.go -trace ..\Tests\trace.log

The result could look like the following:

```diff
Alternatives for fun14,[(2(0),?,P,go-examples\newsReader.go:25)],[(fun14,1)]
+        bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)]
Alternatives for bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)],[(bloomberg32,1)]
-        fun15,[(2(0),?,P,go-examples\newsReader.go:25)]
+        fun14,[(2(0),?,P,go-examples\newsReader.go:25)]
Alternatives for fun03,[(1(0),?,P,go-examples\newsReader.go:21)],[(fun03,1)]
+        reuters20,[(1(0),!,P,go-examples\newsReader.go:7)]
Alternatives for reuters20,[(1(0),!,P,go-examples\newsReader.go:7)],[(reuters20,1)]
-        fun06,[(1(0),?,P,go-examples\newsReader.go:21)]
+        fun03,[(1(0),?,P,go-examples\newsReader.go:21)]
Alternatives for fun15,[(2(0),?,P,go-examples\newsReader.go:25)],[(fun15,1)]
-        bloomberg32,[(2(0),!,P,go-examples\newsReader.go:12)]
Alternatives for fun06,[(1(0),?,P,go-examples\newsReader.go:21)],[(fun06,1)]
-        reuters20,[(1(0),!,P,go-examples\newsReader.go:7)]
Alternatives for newsReader41,[(4(0),?,P,go-examples\newsReader.go:28)],[(newsReader41,1)]
+        fun03,[(4(0),!,P,go-examples\newsReader.go:21)]
Alternatives for fun03,[(4(0),!,P,go-examples\newsReader.go:21)],[(fun03,3)(reuters20,2)]
+        newsReader41,[(4(0),?,P,go-examples\newsReader.go:28)]
Alternatives for main,[(3(0),?,P,go-examples\newsReader.go:28)],[(main,1)]
+        fun14,[(3(0),!,P,go-examples\newsReader.go:25)]
Alternatives for fun14,[(3(0),!,P,go-examples\newsReader.go:25)],[(fun14,3)(bloomberg32,2)]
+        main,[(3(0),?,P,go-examples\newsReader.go:28)]
```