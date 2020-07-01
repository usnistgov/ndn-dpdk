package runningstat

/*
#include "../../csrc/core/running-stat.h"
*/
import "C"
import (
	"unsafe"
)

// RunningStat collects statistics and allows computing min, max, mean, and variance.
// Algorithm comes from https://www.johndcook.com/blog/standard_deviation/ .
type RunningStat struct {
	v runningStat
}

// New constructs a new RunningStat instance in Go memory.
func New() (s *RunningStat) {
	s = new(RunningStat)
	s.Clear(true)
	return s
}

// FromPtr converts *C.RunningStat to RunningStat.
func FromPtr(ptr unsafe.Pointer) (s *RunningStat) {
	return (*RunningStat)(ptr)
}

// Clear deletes collects data.
func (s *RunningStat) Clear(enableMinMax bool) {
	C.RunningStat_Clear(s.v.ptr(), C.bool(enableMinMax))
}

// SetSampleRate changes sample rate be once every 2^q inputs.
func (s *RunningStat) SetSampleRate(q int) {
	C.RunningStat_SetSampleRate(s.v.ptr(), C.int(q))
}

// Push adds an input.
func (s *RunningStat) Push(x float64) {
	C.RunningStat_Push(s.v.ptr(), C.double(x))
}

// Read returns current counters as Snapshot.
func (s *RunningStat) Read() (o Snapshot) {
	o.v = s.v
	return o
}

func (s *runningStat) ptr() *C.RunningStat {
	return (*C.RunningStat)(unsafe.Pointer(s))
}
