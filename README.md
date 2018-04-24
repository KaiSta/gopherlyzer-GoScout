# Gopherlyzer-GoScout v2

Prototype implementation of our analysis described in [Two-Phase Dynamic Analysis of Message-Passing Go Programs based on Vector Clocks](http://www.home.hs-karlsruhe.de/~suma0002/)

## Description

Understanding the run-time behavior of concurrent programs is a challenging task.
A popular approach is to establish a happens-before relation via vector clocks.
Thus, we can identify bugs and performance bottlenecks, for example,
by checking if two conflicting may happen concurrently.
We employ a two-phase method to derive vector clock information for a wide range of concurrency features that includes all of the message-passing features in Go.
The first instrumentation phase yields a run-time trace that contains all events that  took place.
The second trace replay phase carried out offline infers vector clock information.
Trace replay operates on thread-local traces.
Thus, we can observe behavior that might result from some alternative schedule.
Our approach is not tied to any specific language.
We have built a prototype for the Go programming language
and provide empirical evidence of the usefulness of our method.

## How to use

We use a small example program to show the process and the results of our prototype. In the following description we assume that the code file is in a folder called tests.

```go
func main() {
    c := make(chan int)
    go func() {
        c <- 1
    }()
    go func() {
        c <- 1
    }()

    <-c
}
```

### Instrumentation

The first step is to instrument the program with our tool in traceInstv2.

```
cd traceInstv2

go run main.go -in tests/main.go -out results/main.go
```

This results in the following instrumented program:

```go
import "*pathTotracer*/tracer"  //needs to be added manually

func main() {
    tracer.Start() //needs to be added manually

	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)
	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(c, "tests\\main.go:6", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "tests\\main.go:6", myTIDCache)
	}()
	tmp2 := tracer.GetWaitSigID()
	tracer.Signal(tmp2, myTIDCache)

	go func() {
		tracer.RegisterThread("fun1")
		tracer.Wait(tmp2, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.PreSend(c, "tests\\main.go:9", myTIDCache)
		c <- struct {
			threadId uint64
			value    int
		}{myTIDCache, 1}
		tracer.PostSend(c, "tests\\main.go:9", myTIDCache)
	}()
	tracer.PreRcv(c, "tests\\main.go:12", myTIDCache)
	tmp3 := <-c
	tracer.PostRcv(c, "tests\\main.go:12", tmp3.threadId, myTIDCache)

    tracer.Stop() //needs to be added manually
}
```

The tracer.Start() and tracer.Stop calls must be added manually at the beginning and the end of the main function. Further the tracer package needs to be imported. The tracer can be found in `/traceInstv2/tracer'.

### Run

Running the instrumented code with

```
go run results/main.go
```

will produce a trace.log file in the same folder.

### Verification

The last step is to run our verification on the produced trace.

```
cd traceVerify

go run main.go -trace results/trace.log
```

The result should look like the following:

```
pending events fun19 1 fun19,[(1(0),!,P,tests\main.go:9)]

Alternatives for (fun19, fun19,[(1(0),!,P,tests\main.go:9)]):
        (fun07, fun07,[(1(0),!,P,tests\main.go:6)]), 1
        (fun19, fun19,[(1(0),!,P,tests\main.go:9)]), 1
Ratio:1/2

Alternatives for thread fun07
Alternatives for thread fun19
Alternatives for thread main1
        For Event main1,[(1(0),?,P,tests\main.go:12)]
                true-fun07,[(1(0),!,P,tests\main.go:6)]
                false-fun19,[(1(0),!,P,tests\main.go:9)]
```

The first part a send and receives that are stuck due to a missing communication partner. This is followed by the found message contentions (MP). In this case it found two senders on the same channel that executed concurrently. The last part shows the alternative match pairs (AC). In case of the main thread it shows that the receive at line 12 could have matched with 2 different threads 'fun07' and 'fun19' and that it matched with thread 'fun07'.


## Older Version

The previous version can be found in the folder `GoScoutv1'.