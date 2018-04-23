package main

func main() {
	c := make(chan struct {
		threadId uint64
		value    int
	})
	tracer.RegisterChan(c, cap(c))
	x := 0
	tracer.WriteAcc(&x, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:5")
	y := 0
	tracer.WriteAcc(&y, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:6")
	z := 0
	tracer.WriteAcc(&z, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:7")
	tracer.ReadAcc(&x, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:8")
	tracer.ReadAcc(&y, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:8")
	tracer.ReadAcc(&y, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:10")
	tracer.ReadAcc(&z, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:12")
	if x > 5 && y < 3 {
		y = 4
		tracer.WriteAcc(&y, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:9")
	} else if y > 2 {
		tracer.RcvPrep(c, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:11")
		tmp1 := <-c
		tracer.RcvCommit(c, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:11", tmp1.threadId)
		z = tmp1.value
		tracer.WriteAcc(&z, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:11")
	} else if ^z > 0 {
		x = 1
		tracer.WriteAcc(&x, "C:\\Users\\stka0001\\Github\\SpeedyGo\\traceInstv2\\tests\\iftest.go:13")
	}
}
