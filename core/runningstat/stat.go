// Package runningstat implements Knuth and Welford's method for computing the standard deviation.
package runningstat

/*
#include "../../csrc/core/running-stat.h"
*/
import "C"
import (
	"math"
	"unsafe"

	binutils "github.com/jfoster/binary-utilities"
	"github.com/zyedidia/generic"
)

// RunningStat collects statistics and allows computing mean and variance.
// Algorithm comes from https://www.johndcook.com/blog/standard_deviation/ .
type RunningStat C.RunningStat

func (s *RunningStat) ptr() *C.RunningStat {
	return (*C.RunningStat)(s)
}

// Init initializes the instance and clears existing data.
// sampleInterval: how often to collect sample, will be adjusted to nearest power of two and truncated between 1 and 2^30.
func (s *RunningStat) Init(sampleInterval int) {
	*s = RunningStat{
		mask: generic.Clamp(C.uint64_t(binutils.NearPowerOfTwo(int64(sampleInterval))), 1, 1<<30) - 1,
	}
}

// Push adds an input.
func (s *RunningStat) Push(x float64) {
	C.RunningStat_Push(s.ptr(), C.double(x))
}

// Read returns current counters as Snapshot.
func (s *RunningStat) Read() Snapshot {
	return newSnapshot(uint64(s.i), uint64(s.n), float64(s.m1), float64(s.m2), false, 0, 0)
}

// FromPtr converts *C.RunningStat to RunningStat.
func FromPtr(ptr unsafe.Pointer) (s *RunningStat) {
	return (*RunningStat)(ptr)
}

type IntStat C.RunningStatI

func (s *IntStat) ptr() *C.RunningStatI {
	return (*C.RunningStatI)(s)
}

// Init initializes the instance and clears existing data.
// sampleInterval: how often to collect sample, will be adjusted to nearest power of two and truncated between 1 and 2^30.
func (s *IntStat) Init(sampleInterval int) {
	(*RunningStat)(unsafe.Pointer(s)).Init(sampleInterval)
	s.min = math.MaxUint64
	s.max = 0
}

// Push adds an input.
func (s *IntStat) Push(x uint64) {
	C.RunningStatI_Push(s.ptr(), C.uint64_t(x))
}

// Read returns current counters as Snapshot.
func (s *IntStat) Read() Snapshot {
	return newSnapshot(uint64(s.s.i), uint64(s.s.n), float64(s.s.m1), float64(s.s.m2), s.s.n > 0, uint64(s.min), uint64(s.max))
}

// IntFromPtr converts *C.RunningStatI to IntStat.
func IntFromPtr(ptr unsafe.Pointer) (s *IntStat) {
	return (*IntStat)(ptr)
}
