package fft

import (
	"math"
	"runtime"
	"sync"

	"../../tracer"
)

var (
	radix2Lock    sync.RWMutex
	radix2Factors = map[int][]complex128{
		4: {complex(1, 0), complex(0, -1), complex(-1, 0), complex(0, 1)},
	}
)

func EnsureRadix2Factors(input_len int) {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&input_len, "radix2.go:36", myTIDCache)
	getRadix2Factors(input_len)
}

func getRadix2Factors(input_len int) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.PreLock(&radix2Lock, "radix2.go:40", myTIDCache)
	radix2Lock.RLock()

	if hasRadix2Factors(input_len) {
		defer func() {
			tracer.PostLock(&radix2Lock, "radix2.go:43", myTIDCache)
			radix2Lock.RUnlock()
		}()
		return radix2Factors[input_len]
	}
	tracer.PostLock(&radix2Lock, "radix2.go:47", myTIDCache)
	radix2Lock.RUnlock()
	tracer.PreLock(&radix2Lock, "radix2.go:48", myTIDCache)
	radix2Lock.Lock()
	defer func() {
		tracer.PostLock(&radix2Lock, "radix2.go:49", myTIDCache)
		radix2Lock.Unlock()
	}()
	tracer.ReadAcc(&input_len, "radix2.go:51", myTIDCache)
	if !hasRadix2Factors(input_len) {
		for i, p := 8, 4; i <= input_len; i, p = i<<1, i {
			tracer.ReadAcc(&radix2Factors, "radix2.go:53", myTIDCache)
			if radix2Factors[i] == nil {
				radix2Factors[i] = make([]complex128, i)
				tracer.WriteAcc(&radix2Factors, "radix2.go:54", myTIDCache)

				for n, j := 0, 0; n < i; n, j = n+2, j+1 {
					radix2Factors[i][n] = radix2Factors[p][j]
					tracer.ReadAcc(&radix2Factors, "radix2.go:57", myTIDCache)
					tracer.ReadAcc(&radix2Factors[p][j], "radix2.go:57", myTIDCache)
					tracer.WriteAcc(&radix2Factors[i][n], "radix2.go:57", myTIDCache)
					tracer.WriteAcc(&radix2Factors, "radix2.go:57", myTIDCache)
				}

				for n := 1; n < i; n += 2 {
					tracer.ReadAcc(&i, "radix2.go:61", myTIDCache)
					tracer.ReadAcc(&n, "radix2.go:61", myTIDCache)
					sin, cos := math.Sincos(-2 * math.Pi / float64(i) * float64(n))
					tracer.WriteAcc(&sin, "radix2.go:61", myTIDCache)
					tracer.WriteAcc(&cos, "radix2.go:61", myTIDCache)
					tracer.ReadAcc(&cos, "radix2.go:62", myTIDCache)
					tracer.ReadAcc(&sin, "radix2.go:62", myTIDCache)
					radix2Factors[i][n] = complex(cos, sin)
					tracer.WriteAcc(&radix2Factors[i][n], "radix2.go:62", myTIDCache)
					tracer.WriteAcc(&radix2Factors, "radix2.go:62", myTIDCache)
				}
			}
		}
	}
	tracer.ReadAcc(&radix2Factors, "radix2.go:68", myTIDCache)
	return radix2Factors[input_len]
}

func hasRadix2Factors(idx int) bool {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&radix2Factors, "radix2.go:72", myTIDCache)
	return radix2Factors[idx] != nil
}

type fft_work struct {
	start, end int
}

func radix2FFT(x []complex128) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "radix2.go:81", myTIDCache)
	lx := len(x)
	tracer.WriteAcc(&lx, "radix2.go:81", myTIDCache)
	tracer.ReadAcc(&lx, "radix2.go:82", myTIDCache)
	factors := getRadix2Factors(lx)
	tracer.WriteAcc(&factors, "radix2.go:82", myTIDCache)
	tracer.ReadAcc(&lx, "radix2.go:84", myTIDCache)
	t := make([]complex128, lx)
	tracer.WriteAcc(&t, "radix2.go:84", myTIDCache)
	tracer.ReadAcc(&x, "radix2.go:85", myTIDCache)
	r := reorderData(x)
	tracer.WriteAcc(&r, "radix2.go:85", myTIDCache)

	var blocks, stage, s_2 int
	tracer.ReadAcc(&lx, "radix2.go:89", myTIDCache)
	jobs := make(chan struct {
		threadId uint64
		value    *fft_work
	}, lx)
	tracer.RegisterChan(jobs, cap(jobs))
	tracer.ReadAcc(&lx, "radix2.go:90", myTIDCache)
	results := make(chan struct {
		threadId uint64
		value    bool
	}, lx)
	tracer.RegisterChan(results, cap(results))

	num_workers := worker_pool_size
	tracer.ReadAcc(&worker_pool_size, "radix2.go:92", myTIDCache)
	tracer.WriteAcc(&num_workers, "radix2.go:92", myTIDCache)
	if (num_workers) == 0 {
		num_workers = runtime.GOMAXPROCS(0)
		tracer.WriteAcc(&num_workers, "radix2.go:94", myTIDCache)
	}
	tracer.ReadAcc(&lx, "radix2.go:97", myTIDCache)
	tracer.ReadAcc(&num_workers, "radix2.go:97", myTIDCache)
	idx_diff := lx / num_workers
	tracer.WriteAcc(&idx_diff, "radix2.go:97", myTIDCache)
	tracer.ReadAcc(&idx_diff, "radix2.go:98", myTIDCache)
	if idx_diff < 2 {
		myTIDCache := tracer.GetGID()
		idx_diff = 2
		tracer.WriteAcc(&idx_diff, "radix2.go:99", myTIDCache)
	}

	worker := func() {
		myTIDCache := tracer.GetGID()
		for work := range jobs {
			tracer.PreRcv(jobs, "radix2.go:103", myTIDCache)
			tracer.PostRcv(jobs, "radix2.go:103", work.threadId, myTIDCache)
			for nb := work.value.start; nb < work.value.end; nb += stage {
				tracer.ReadAcc(&stage, "radix2.go:105", myTIDCache)
				if stage != 2 {
					tracer.ReadAcc(&s_2, "radix2.go:106", myTIDCache)
					for j := 0; j < s_2; j++ {
						tracer.ReadAcc(&nb, "radix2.go:107", myTIDCache)
						idx := j + nb
						tracer.WriteAcc(&idx, "radix2.go:107", myTIDCache)
						tracer.ReadAcc(&idx, "radix2.go:108", myTIDCache)
						tracer.ReadAcc(&s_2, "radix2.go:108", myTIDCache)
						idx2 := idx + s_2
						tracer.WriteAcc(&idx2, "radix2.go:108", myTIDCache)
						tracer.ReadAcc(&r[idx], "radix2.go:109", myTIDCache)
						ridx := r[idx]
						tracer.WriteAcc(&ridx, "radix2.go:109", myTIDCache)

						tracer.ReadAcc(&r[idx2], "radix2.go:110", myTIDCache)
						tracer.ReadAcc(&factors[blocks*j], "radix2.go:110", myTIDCache)
						w_n := r[idx2] * factors[blocks*j]
						tracer.WriteAcc(&w_n, "radix2.go:110", myTIDCache)

						tracer.ReadAcc(&ridx, "radix2.go:111", myTIDCache)
						tracer.ReadAcc(&w_n, "radix2.go:111", myTIDCache)
						t[idx] = ridx + w_n
						tracer.WriteAcc(&t[idx], "radix2.go:111", myTIDCache)
						tracer.ReadAcc(&ridx, "radix2.go:112", myTIDCache)
						tracer.ReadAcc(&w_n, "radix2.go:112", myTIDCache)
						t[idx2] = ridx - w_n
						tracer.WriteAcc(&t[idx2], "radix2.go:111", myTIDCache)
					}
				} else {
					tracer.ReadAcc(&nb, "radix2.go:115", myTIDCache)
					n1 := nb + 1
					tracer.WriteAcc(&n1, "radix2.go:115", myTIDCache)
					tracer.ReadAcc(&r[nb], "radix2.go:116", myTIDCache)
					rn := r[nb]
					tracer.WriteAcc(&rn, "radix2.go:116", myTIDCache)
					tracer.ReadAcc(&r[n1], "radix2.go:117", myTIDCache)
					rn1 := r[n1]
					tracer.WriteAcc(&rn1, "radix2.go:117", myTIDCache)
					t[nb] = rn + rn1
					tracer.WriteAcc(&t[nb], "radix2.go:118", myTIDCache)
					t[n1] = rn - rn1
					tracer.WriteAcc(&t[n1], "radix2.go:119", myTIDCache)
				}
			}
			tracer.PreSend(results, "radix2.go:123", myTIDCache)
			results <- struct {
				threadId uint64
				value    bool
			}{myTIDCache, true}
			tracer.PostSend(results, "radix2.go:123", myTIDCache)
		}
	}
	tracer.WriteAcc(&worker, "radix2.go:102", myTIDCache)

	for i := 0; i < num_workers; i++ {
		tmp1 := tracer.GetWaitSigID()
		tracer.Signal(tmp1, myTIDCache)
		go func() {
			tracer.RegisterThread("worker0")
			tracer.Wait(tmp1, tracer.GetGID())
			worker()
		}()
	}
	defer func() {
		tracer.PreClose(jobs, "radix2.go:130", myTIDCache)
		close(jobs)
		tracer.PostClose(jobs, "radix2.go:130", myTIDCache)
	}()

	for stage = 2; stage <= lx; stage <<= 1 {
		tracer.ReadAcc(&lx, "radix2.go:133", myTIDCache)
		tracer.ReadAcc(&stage, "radix2.go:133", myTIDCache)
		blocks = lx / stage
		tracer.WriteAcc(&blocks, "radix2.go:133", myTIDCache)
		tracer.ReadAcc(&stage, "radix2.go:134", myTIDCache)
		s_2 = stage / 2
		tracer.WriteAcc(&s_2, "radix2.go:134", myTIDCache)
		workers_spawned := 0
		tracer.WriteAcc(&workers_spawned, "radix2.go:135", myTIDCache)

		for start, end := 0, stage; ; {
			tracer.ReadAcc(&end, "radix2.go:138", myTIDCache)
			tracer.ReadAcc(&start, "radix2.go:138", myTIDCache)
			tracer.ReadAcc(&idx_diff, "radix2.go:138", myTIDCache)
			tracer.ReadAcc(&end, "radix2.go:138", myTIDCache)
			tracer.ReadAcc(&lx, "radix2.go:138", myTIDCache)
			if end-start >= idx_diff || end == lx {
				tracer.WriteAcc(&workers_spawned, "radix2.go:139", myTIDCache)
				workers_spawned++
				tracer.PreSend(jobs, "radix2.go:140", myTIDCache)
				jobs <- struct {
					threadId uint64
					value    *fft_work
				}{myTIDCache, &fft_work{start, end}}
				tracer.PostSend(jobs, "radix2.go:140", myTIDCache)
				tracer.ReadAcc(&end, "radix2.go:142", myTIDCache)
				tracer.ReadAcc(&lx, "radix2.go:142", myTIDCache)
				if end == lx {
					break
				}

				start = end
				tracer.ReadAcc(&end, "radix2.go:146", myTIDCache)
				tracer.WriteAcc(&start, "radix2.go:146", myTIDCache)
			}

			end += stage
			tracer.ReadAcc(&stage, "radix2.go:149", myTIDCache)
			tracer.WriteAcc(&end, "radix2.go:149", myTIDCache)
		}

		for n := 0; n < workers_spawned; n++ {
			tracer.PreRcv(results, "radix2.go:153", myTIDCache)
			tmp2 := <-results
			tracer.PostRcv(results, "radix2.go:153", tmp2.threadId, myTIDCache)
		}

		r, t = t, r
		tracer.ReadAcc(&t, "radix2.go:156", myTIDCache)
		tracer.ReadAcc(&r, "radix2.go:156", myTIDCache)
		tracer.WriteAcc(&r, "radix2.go:156", myTIDCache)
		tracer.WriteAcc(&t, "radix2.go:156", myTIDCache)
	}

	return r
}

func reorderData(x []complex128) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "radix2.go:164", myTIDCache)
	lx := uint(len(x))
	tracer.WriteAcc(&lx, "radix2.go:164", myTIDCache)
	tracer.ReadAcc(&lx, "radix2.go:165", myTIDCache)
	r := make([]complex128, lx)
	tracer.WriteAcc(&r, "radix2.go:165", myTIDCache)
	tracer.ReadAcc(&lx, "radix2.go:166", myTIDCache)
	s := log2(lx)
	tracer.WriteAcc(&s, "radix2.go:166", myTIDCache)

	var n uint
	for ; n < lx; n++ {
		r[reverseBits(n, s)] = x[n]
		tracer.ReadAcc(&x[n], "radix2.go:170", myTIDCache)
	}

	return r
}

func log2(v uint) uint {
	myTIDCache := tracer.GetGID()
	var r uint

	for v >>= 1; v != 0; v >>= 1 {
		tracer.WriteAcc(&r, "radix2.go:182", myTIDCache)
		r++
	}

	return r
}

func reverseBits(v, s uint) uint {
	myTIDCache := tracer.GetGID()
	var r uint
	tracer.ReadAcc(&v, "radix2.go:195", myTIDCache)
	r = v & 1
	tracer.WriteAcc(&r, "radix2.go:195", myTIDCache)
	tracer.WriteAcc(&s, "radix2.go:196", myTIDCache)
	s--

	for v >>= 1; v != 0; v >>= 1 {
		r <<= 1
		tracer.WriteAcc(&r, "radix2.go:199", myTIDCache)
		tracer.ReadAcc(&v, "radix2.go:200", myTIDCache)
		r |= v & 1
		tracer.WriteAcc(&r, "radix2.go:200", myTIDCache)
		tracer.WriteAcc(&s, "radix2.go:201", myTIDCache)
		s--
	}

	return r << s
}
