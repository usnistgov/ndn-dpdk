package running_stat

/*
#include "running-stat.h"
*/
import "C"
import (
	"math"
	"unsafe"
)

// Compute min, max, mean, and variance.
// https://www.johndcook.com/blog/standard_deviation/
type RunningStat struct {
	c *C.RunningStat
}

func New() (s RunningStat) {
	s.c = new(C.RunningStat)
	s.Clear(true)
	return s
}

func FromPtr(ptr unsafe.Pointer) (s RunningStat) {
	s.c = (*C.RunningStat)(ptr)
	return s
}

func (s RunningStat) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(s.c)
}

func (s RunningStat) Clear(enableMinMax bool) {
	C.RunningStat_Clear(s.c, C.bool(enableMinMax))
}

// Set sample rate to once every 2^q inputs.
func (s RunningStat) SetSampleRate(q int) {
	C.RunningStat_SetSampleRate(s.c, C.int(q))
}

// Update with an input.
func (s RunningStat) Push(x float64) {
	C.RunningStat_Push(s.c, C.double(x))
}

func (s RunningStat) Len() int {
	return int(s.c.n)
}

// Get number of samples.
func (s RunningStat) Len64() uint64 {
	return uint64(s.c.n)
}

func (s RunningStat) Min() float64 {
	if s.c.n == 0 {
		return math.NaN()
	}
	return float64(s.c.min)
}

func (s RunningStat) Max() float64 {
	if s.c.n == 0 {
		return math.NaN()
	}
	return float64(s.c.max)
}

func (s RunningStat) Mean() float64 {
	if s.c.n == 0 {
		return math.NaN()
	}
	return float64(s.c.m1)
}

func (s RunningStat) Variance() float64 {
	if s.c.n <= 1 {
		return math.NaN()
	}
	return float64(s.c.m2) / float64(s.c.n-1)
}

func (s RunningStat) Stdev() float64 {
	return math.Sqrt(s.Variance())
}

func Combine(a, b RunningStat) (c RunningStat) {
	c.c = new(C.RunningStat)
	c.c.min = C.fmin(a.c.min, b.c.min)
	c.c.max = C.fmax(a.c.max, b.c.max)
	c.c.n = a.c.n + b.c.n
	delta := b.c.m1 - a.c.m1
	delta2 := delta * delta
	c.c.m1 = (C.double(a.c.n)*a.c.m1 + C.double(b.c.n)*b.c.m1) / C.double(c.c.n)
	c.c.m2 = a.c.m2 + b.c.m2 + delta2*C.double(a.c.n)*C.double(b.c.n)/C.double(c.c.n)
	return c
}
