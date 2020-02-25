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
	v runningStat
}

func New() (s *RunningStat) {
	s = new(RunningStat)
	s.Clear(true)
	return s
}

func FromPtr(ptr unsafe.Pointer) (s *RunningStat) {
	return (*RunningStat)(ptr)
}

func (s *RunningStat) Clear(enableMinMax bool) {
	C.RunningStat_Clear(s.v.getPtr(), C.bool(enableMinMax))
}

// Set sample rate to once every 2^q inputs.
func (s *RunningStat) SetSampleRate(q int) {
	C.RunningStat_SetSampleRate(s.v.getPtr(), C.int(q))
}

// Update with an input.
func (s *RunningStat) Push(x float64) {
	C.RunningStat_Push(s.v.getPtr(), C.double(x))
}

// Combine state with another instance.
func (s *RunningStat) Combine(other *RunningStat) {
	s.v.combine(other.v)
}

// Read counters as snapshot.
func (s *RunningStat) Read() (o Snapshot) {
	o.v = s.v
	return o
}

func (s *runningStat) getPtr() *C.RunningStat {
	return (*C.RunningStat)(unsafe.Pointer(s))
}

func (a *runningStat) combine(b runningStat) {
	if a.I == 0 {
		*a = b
		return
	} else if b.I == 0 {
		return
	}
	a.I += b.I
	a.Min = math.Min(a.Min, b.Min)
	a.Max = math.Max(a.Max, b.Max)
	aN := float64(a.N)
	bN := float64(b.N)
	a.N += b.N
	cN := float64(a.N)
	delta := b.M1 - a.M1
	delta2 := delta * delta
	a.M1 = (aN*a.M1 + bN*b.M1) / cN
	a.M2 += b.M2 + delta2*aN*bN/cN
}
