package fft

import (
	"../../tracer"
	"../dsputils"
)

func FFTReal(x []float64) []complex128 {
	return FFT(dsputils.ToComplex(x))
}

func IFFTReal(x []float64) []complex128 {
	return IFFT(dsputils.ToComplex(x))
}

func IFFT(x []complex128) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "fft.go:36", myTIDCache)
	lx := len(x)
	tracer.WriteAcc(&lx, "fft.go:36", myTIDCache)
	tracer.ReadAcc(&lx, "fft.go:37", myTIDCache)
	r := make([]complex128, lx)
	tracer.WriteAcc(&r, "fft.go:37", myTIDCache)

	r[0] = x[0]
	tracer.ReadAcc(&x[0], "fft.go:40", myTIDCache)
	tracer.WriteAcc(&r[0], "fft.go:40", myTIDCache)
	for i := 1; i < lx; i++ {
		r[i] = x[lx-i]
		tracer.ReadAcc(&x[lx-i], "fft.go:42", myTIDCache)
		tracer.WriteAcc(&r[i], "fft.go:42", myTIDCache)
	}
	tracer.ReadAcc(&r, "fft.go:45", myTIDCache)
	r = FFT(r)
	tracer.WriteAcc(&r, "fft.go:45", myTIDCache)
	tracer.ReadAcc(&lx, "fft.go:47", myTIDCache)
	N := complex(float64(lx), 0)
	tracer.WriteAcc(&N, "fft.go:47", myTIDCache)
	for n := range r {
		r[n] /= N
		tracer.ReadAcc(&N, "fft.go:49", myTIDCache)
		tracer.WriteAcc(&r[n], "fft.go:49", myTIDCache)
	}
	return r
}

func Convolve(x, y []complex128) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "fft.go:56", myTIDCache)
	tracer.ReadAcc(&y, "fft.go:56", myTIDCache)
	if len(x) != len(y) {
		panic("arrays not of equal size")
	}
	tracer.ReadAcc(&x, "fft.go:60", myTIDCache)
	fft_x := FFT(x)
	tracer.WriteAcc(&fft_x, "fft.go:60", myTIDCache)
	tracer.ReadAcc(&y, "fft.go:61", myTIDCache)
	fft_y := FFT(y)
	tracer.WriteAcc(&fft_y, "fft.go:61", myTIDCache)
	tracer.ReadAcc(&x, "fft.go:63", myTIDCache)
	r := make([]complex128, len(x))
	tracer.WriteAcc(&r, "fft.go:63", myTIDCache)
	for i := 0; i < len(r); i++ {
		tracer.ReadAcc(&fft_x[i], "fft.go:65", myTIDCache)
		tracer.ReadAcc(&fft_y[i], "fft.go:65", myTIDCache)
		r[i] = fft_x[i] * fft_y[i]
		tracer.WriteAcc(&r[i], "fft.go:65", myTIDCache)
	}

	return IFFT(r)
}

func FFT(x []complex128) []complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "fft.go:73", myTIDCache)
	lx := len(x)
	tracer.WriteAcc(&lx, "fft.go:73", myTIDCache)
	tracer.ReadAcc(&lx, "fft.go:76", myTIDCache)
	if lx <= 1 {
		tracer.ReadAcc(&lx, "fft.go:77", myTIDCache)
		r := make([]complex128, lx)
		tracer.WriteAcc(&r, "fft.go:77", myTIDCache)
		tracer.ReadAcc(&r, "fft.go:78", myTIDCache)
		tracer.ReadAcc(&x, "fft.go:78", myTIDCache)
		copy(r, x)
		return r
	}

	if dsputils.IsPowerOf2(lx) {
		return radix2FFT(x)
	}

	return bluesteinFFT(x)
}

var (
	worker_pool_size = 0
)

func SetWorkerPoolSize(n int) {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&n, "fft.go:96", myTIDCache)
	if n < 0 {
		n = 0
		tracer.WriteAcc(&n, "fft.go:97", myTIDCache)
	}

	worker_pool_size = n
	tracer.ReadAcc(&n, "fft.go:100", myTIDCache)
	tracer.WriteAcc(&worker_pool_size, "fft.go:100", myTIDCache)
}

func FFT2Real(x [][]float64) [][]complex128 {
	return FFT2(dsputils.ToComplex2(x))
}

func FFT2(x [][]complex128) [][]complex128 {
	return computeFFT2(x, FFT)
}

func IFFT2Real(x [][]float64) [][]complex128 {
	return IFFT2(dsputils.ToComplex2(x))
}

func IFFT2(x [][]complex128) [][]complex128 {
	return computeFFT2(x, IFFT)
}

func computeFFT2(x [][]complex128, fftFunc func([]complex128) []complex128) [][]complex128 {
	myTIDCache := tracer.GetGID()
	tracer.ReadAcc(&x, "fft.go:124", myTIDCache)
	rows := len(x)
	tracer.WriteAcc(&rows, "fft.go:124", myTIDCache)
	tracer.ReadAcc(&rows, "fft.go:125", myTIDCache)
	if rows == 0 {
		panic("empty input array")
	}
	tracer.ReadAcc(&x[0], "fft.go:129", myTIDCache)
	cols := len(x[0])
	tracer.WriteAcc(&cols, "fft.go:129", myTIDCache)
	tracer.ReadAcc(&rows, "fft.go:130", myTIDCache)
	r := make([][]complex128, rows)
	tracer.WriteAcc(&r, "fft.go:130", myTIDCache)
	for i := 0; i < rows; i++ {
		tracer.ReadAcc(&x[i], "fft.go:132", myTIDCache)
		tracer.ReadAcc(&cols, "fft.go:132", myTIDCache)
		if len(x[i]) != cols {
			panic("ragged input array")
		}
		tracer.ReadAcc(&cols, "fft.go:135", myTIDCache)
		r[i] = make([]complex128, cols)
		tracer.WriteAcc(&r[i], "fft.go:135", myTIDCache)
	}

	for i := 0; i < cols; i++ {
		tracer.ReadAcc(&rows, "fft.go:139", myTIDCache)
		t := make([]complex128, rows)
		tracer.WriteAcc(&t, "fft.go:139", myTIDCache)
		for j := 0; j < rows; j++ {
			t[j] = x[j][i]
			tracer.ReadAcc(&x[j][i], "fft.go:141", myTIDCache)
			tracer.WriteAcc(&t[j], "fft.go:141", myTIDCache)
		}

		for n, v := range fftFunc(t) {
			r[n][i] = v
			tracer.ReadAcc(&v, "fft.go:145", myTIDCache)
			tracer.WriteAcc(&r[n][i], "fft.go:145", myTIDCache)
		}
	}

	for n, v := range r {
		tracer.ReadAcc(&v, "fft.go:150", myTIDCache)
		r[n] = fftFunc(v)
		tracer.WriteAcc(&r[n], "fft.go:150", myTIDCache)
	}

	return r
}

func FFTN(m *dsputils.Matrix) *dsputils.Matrix {
	return computeFFTN(m, FFT)
}

func IFFTN(m *dsputils.Matrix) *dsputils.Matrix {
	return computeFFTN(m, IFFT)
}

func computeFFTN(m *dsputils.Matrix, fftFunc func([]complex128) []complex128) *dsputils.Matrix {
	myTIDCache := tracer.GetGID()
	dims := m.Dimensions()
	tracer.WriteAcc(&dims, "fft.go:167", myTIDCache)
	t := m.Copy()
	tracer.WriteAcc(&t, "fft.go:168", myTIDCache)
	tracer.ReadAcc(&dims, "fft.go:169", myTIDCache)
	r := dsputils.MakeEmptyMatrix(dims)
	tracer.WriteAcc(&r, "fft.go:169", myTIDCache)

	for n := range dims {
		dims[n] -= 1
		tracer.WriteAcc(&dims[n], "fft.go:172", myTIDCache)
	}

	for n := range dims {
		tracer.ReadAcc(&dims, "fft.go:176", myTIDCache)
		d := make([]int, len(dims))
		tracer.WriteAcc(&d, "fft.go:176", myTIDCache)
		tracer.ReadAcc(&d, "fft.go:177", myTIDCache)
		tracer.ReadAcc(&dims, "fft.go:177", myTIDCache)
		copy(d, dims)
		d[n] = -1
		tracer.WriteAcc(&d[n], "fft.go:178", myTIDCache)

		for {
			tracer.ReadAcc(&d, "fft.go:181", myTIDCache)
			tracer.ReadAcc(&d, "fft.go:181", myTIDCache)
			r.SetDim(fftFunc(t.Dim(d)), d)
			tracer.ReadAcc(&d, "fft.go:183", myTIDCache)
			tracer.ReadAcc(&dims, "fft.go:183", myTIDCache)
			if !decrDim(d, dims) {
				break
			}
		}

		r, t = t, r
		tracer.ReadAcc(&t, "fft.go:188", myTIDCache)
		tracer.ReadAcc(&r, "fft.go:188", myTIDCache)
		tracer.WriteAcc(&r, "fft.go:188", myTIDCache)
		tracer.WriteAcc(&t, "fft.go:188", myTIDCache)
	}

	return t
}

func decrDim(x, d []int) bool {
	myTIDCache := tracer.GetGID()
	for n, v := range x {
		tracer.ReadAcc(&v, "fft.go:199", myTIDCache)
		tracer.ReadAcc(&v, "fft.go:201", myTIDCache)
		if v == -1 {
			continue
		} else if v == 0 {
			i := n
			tracer.ReadAcc(&n, "fft.go:202", myTIDCache)
			tracer.WriteAcc(&i, "fft.go:202", myTIDCache)

			for ; i < len(x); i++ {
				tracer.ReadAcc(&x[i], "fft.go:205", myTIDCache)
				if x[i] == -1 {
					continue
				} else if x[i] == 0 {
					x[i] = d[i]
					tracer.ReadAcc(&d[i], "fft.go:208", myTIDCache)
					tracer.WriteAcc(&x[i], "fft.go:208", myTIDCache)
				} else {
					x[i] -= 1
					tracer.WriteAcc(&x[i], "fft.go:210", myTIDCache)
					return true
				}
			}

			return false
		} else {
			x[n] -= 1
			tracer.WriteAcc(&x[n], "fft.go:218", myTIDCache)
			return true
		}
	}

	return false
}
