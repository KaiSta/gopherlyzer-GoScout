package main

func main() {
	myTIDCache := tracer.GetGID()
	tracer.RegisterThread("main")
	x := 0
	tracer.WriteAcc(&x, "ex6.go:4", myTIDCache)
	tmp1 := tracer.GetWaitSigID()
	tracer.Signal(tmp1, myTIDCache)

	go func() {
		tracer.RegisterThread("fun0")
		tracer.Wait(tmp1, tracer.GetGID())
		myTIDCache := tracer.GetGID()
		tracer.WriteAcc(&x, "ex6.go:7", myTIDCache)
		x++
	}()
	tracer.ReadAcc(&x, "ex6.go:10", myTIDCache)
	print(x)
}