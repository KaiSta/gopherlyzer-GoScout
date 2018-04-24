package main

import (
	"../../go-dsp/fft"
	"../../tracer"
)

import (
	"math"
	"math/cmplx"
	"runtime"

	"../../go-dsp/dsputils"
)

const (
	sqrt2_2 = math.Sqrt2 / 2
)

type fftTest struct {
	in  []float64
	out []complex128
}

var fftTests = []fftTest{
	// impulse responses
	{
		[]float64{1},
		[]complex128{complex(1, 0)},
	},
	{
		[]float64{1, 0},
		[]complex128{complex(1, 0), complex(1, 0)},
	},
	{
		[]float64{1, 0, 0, 0},
		[]complex128{complex(1, 0), complex(1, 0), complex(1, 0), complex(1, 0)},
	},
	{
		[]float64{1, 0, 0, 0, 0, 0, 0, 0},
		[]complex128{
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0)},
	},

	// shifted impulse response
	{
		[]float64{0, 1},
		[]complex128{complex(1, 0), complex(-1, 0)},
	},
	{
		[]float64{0, 1, 0, 0},
		[]complex128{complex(1, 0), complex(0, -1), complex(-1, 0), complex(0, 1)},
	},
	{
		[]float64{0, 1, 0, 0, 0, 0, 0, 0},
		[]complex128{
			complex(1, 0),
			complex(sqrt2_2, -sqrt2_2),
			complex(0, -1),
			complex(-sqrt2_2, -sqrt2_2),
			complex(-1, 0),
			complex(-sqrt2_2, sqrt2_2),
			complex(0, 1),
			complex(sqrt2_2, sqrt2_2)},
	},

	// other
	{
		[]float64{1, 2, 3, 4},
		[]complex128{
			complex(10, 0),
			complex(-2, 2),
			complex(-2, 0),
			complex(-2, -2)},
	},
	{
		[]float64{1, 3, 5, 7},
		[]complex128{
			complex(16, 0),
			complex(-4, 4),
			complex(-4, 0),
			complex(-4, -4)},
	},
	{
		[]float64{1, 2, 3, 4, 5, 6, 7, 8},
		[]complex128{
			complex(36, 0),
			complex(-4, 9.65685425),
			complex(-4, 4),
			complex(-4, 1.65685425),
			complex(-4, 0),
			complex(-4, -1.65685425),
			complex(-4, -4),
			complex(-4, -9.65685425)},
	},

	// non power of 2 lengths
	{
		[]float64{1, 0, 0, 0, 0},
		[]complex128{
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0),
			complex(1, 0)},
	},
	{
		[]float64{1, 2, 3},
		[]complex128{
			complex(6, 0),
			complex(-1.5, 0.8660254),
			complex(-1.5, -0.8660254)},
	},
	{
		[]float64{1, 1, 1},
		[]complex128{
			complex(3, 0),
			complex(0, 0),
			complex(0, 0)},
	},
}

type fft2Test struct {
	in  [][]float64
	out [][]complex128
}

var fft2Tests = []fft2Test{
	{
		[][]float64{{1, 2, 3}, {3, 4, 5}},
		[][]complex128{
			{complex(18, 0), complex(-3, 1.73205081), complex(-3, -1.73205081)},
			{complex(-6, 0), complex(0, 0), complex(0, 0)}},
	},
	{
		[][]float64{{0.1, 0.2, 0.3, 0.4, 0.5}, {1, 2, 3, 4, 5}, {3, 2, 1, 0, -1}},
		[][]complex128{
			{complex(21.5, 0), complex(-0.25, 0.34409548), complex(-0.25, 0.08122992), complex(-0.25, -0.08122992), complex(-0.25, -0.34409548)},
			{complex(-8.5, -8.66025404), complex(5.70990854, 4.6742225), complex(1.15694356, 4.41135694), complex(-1.65694356, 4.24889709), complex(-6.20990854, 3.98603154)},
			{complex(-8.5, 8.66025404), complex(-6.20990854, -3.98603154), complex(-1.65694356, -4.24889709), complex(1.15694356, -4.41135694), complex(5.70990854, -4.6742225)}},
	},
}

type fftnTest struct {
	in  []float64
	dim []int
	out []complex128
}

var fftnTests = []fftnTest{
	{
		[]float64{4, 2, 3, 8, 5, 6, 7, 2, 13, 24, 13, 17},
		[]int{2, 2, 3},
		[]complex128{
			complex(104, 0), complex(12.5, 14.72243186), complex(12.5, -14.72243186),
			complex(-42, 0), complex(-10.5, 6.06217783), complex(-10.5, -6.06217783),

			complex(-48, 0), complex(-4.5, -11.25833025), complex(-4.5, 11.25833025),
			complex(22, 0), complex(8.5, -6.06217783), complex(8.5, 6.06217783)},
	},
}

type reverseBitsTest struct {
	in  uint
	sz  uint
	out uint
}

var reverseBitsTests = []reverseBitsTest{
	{0, 1, 0},
	{1, 2, 2},
	{1, 4, 8},
	{2, 4, 4},
	{3, 4, 12},
}

func TestFFT() {
	for _, ft := range fftTests {
		v := fft.FFTReal(ft.in)
		if !dsputils.PrettyCloseC(v, ft.out) {
			//t.Error("FFT error\ninput:", ft.in, "\noutput:", v, "\nexpected:", ft.out)
		}

		vi := fft.IFFT(ft.out)
		if !dsputils.PrettyCloseC(vi, dsputils.ToComplex(ft.in)) {
			//t.Error("IFFT error\ninput:", ft.out, "\noutput:", vi, "\nexpected:", dsputils.ToComplex(ft.in))
		}
	}
}

func TestFFT2() {
	for _, ft := range fft2Tests {
		v := fft.FFT2Real(ft.in)
		if !dsputils.PrettyClose2(v, ft.out) {
			//t.Error("FFT2 error\ninput:", ft.in, "\noutput:", v, "\nexpected:", ft.out)
		}

		vi := fft.IFFT2(ft.out)
		if !dsputils.PrettyClose2(vi, dsputils.ToComplex2(ft.in)) {
			//	t.Error("IFFT2 error\ninput:", ft.out, "\noutput:", vi, "\nexpected:", dsputils.ToComplex2(ft.in))
		}
	}
}

func TestFFTN() {
	for _, ft := range fftnTests {
		m := dsputils.MakeMatrix(dsputils.ToComplex(ft.in), ft.dim)
		o := dsputils.MakeMatrix(ft.out, ft.dim)
		v := fft.FFTN(m)
		if !v.PrettyClose(o) {
			//	t.Error("FFTN error\ninput:", m, "\noutput:", v, "\nexpected:", o)
		}

		vi := fft.IFFTN(o)
		if !vi.PrettyClose(m) {
			//	t.Error("IFFTN error\ninput:", o, "\noutput:", vi, "\nexpected:", m)
		}
	}
}

func TestFFTMulti() {
	N := 1 << 8
	a := make([]complex128, N)
	for i := 0; i < N; i++ {
		a[i] = complex(float64(i)/float64(N), 0)
	}

	fft.FFT(a)
}

// run with: go test -test.bench="."
func BenchmarkFFT() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	N := 1 << 20
	a := make([]complex128, N)
	for i := 0; i < N; i++ {
		a[i] = complex(float64(i)/float64(N), 0)
	}

	fft.EnsureRadix2Factors(N)

	for i := 0; i < 5; i++ {
		fft.FFT(a)
	}
}

// This example is adapted from Richard Lyon's "Understanding Digital Signal Processing," section 3.1.1.
func ExampleFFTReal() {
	numSamples := 8

	// Equation 3-10.
	x := func(n int) float64 {
		wave0 := math.Sin(2.0 * math.Pi * float64(n) / 8.0)
		wave1 := 0.5 * math.Sin(2*math.Pi*float64(n)/4.0+3.0*math.Pi/4.0)
		return wave0 + wave1
	}

	// Discretize our function by sampling at 8 points.
	a := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		a[i] = x(i)
	}

	X := fft.FFTReal(a)

	// Print the magnitude and phase at each frequency.
	for i := 0; i < numSamples; i++ {
		r, θ := cmplx.Polar(X[i])
		θ *= 360.0 / (2 * math.Pi)
		if dsputils.Float64Equal(r, 0) {
			θ = 0 // (When the magnitude is close to 0, the angle is meaningless)
		}
		//	fmt.Printf("X(%d) = %.1f ∠ %.1f°\n", i, r, θ)
	}

	// Output:
	// X(0) = 0.0 ∠ 0.0°
	// X(1) = 4.0 ∠ -90.0°
	// X(2) = 2.0 ∠ 45.0°
	// X(3) = 0.0 ∠ 0.0°
	// X(4) = 0.0 ∠ 0.0°
	// X(5) = 0.0 ∠ 0.0°
	// X(6) = 2.0 ∠ -45.0°
	// X(7) = 4.0 ∠ 90.0°
}

func main() {
	tracer.Start()
	tracer.RegisterThread("main")

	TestFFT()
	TestFFT2()
	TestFFTN()
	TestFFTMulti()
	//BenchmarkFFT()
	ExampleFFTReal()

	tracer.Stop()
}
