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
	return s
}

func FromPtr(ptr unsafe.Pointer) (s RunningStat) {
	s.c = (*C.RunningStat)(ptr)
	return s
}

func (s RunningStat) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(s.c)
}

func (s RunningStat) Push(x float64) {
	C.RunningStat_Push(s.c, C.double(x))
}

func (s RunningStat) Len() int {
	return int(s.c.n)
}

func (s RunningStat) Len64() uint64 {
	return uint64(s.c.n)
}

func (s RunningStat) Min() float64 {
	return float64(s.c.min)
}

func (s RunningStat) Max() float64 {
	return float64(s.c.max)
}

func (s RunningStat) Mean() float64 {
	if s.c.n == 0 {
		return 0.0
	}
	return float64(s.c.newM)
}

func (s RunningStat) Variance() float64 {
	if s.c.n <= 1 {
		return 0.0
	}
	return float64(s.c.newS) / float64(s.c.n-1)
}

func (s RunningStat) Stdev() float64 {
	return math.Sqrt(s.Variance())
}
