package running_stat

/*
#include "running-stat.h"
*/
import "C"
import (
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

// Read counters as snapshot.
func (s *RunningStat) Read() (o Snapshot) {
	o.v = s.v
	return o
}

func (s *runningStat) getPtr() *C.RunningStat {
	return (*C.RunningStat)(unsafe.Pointer(s))
}
