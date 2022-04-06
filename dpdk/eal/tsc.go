package eal

/*
#include "../../csrc/dpdk/tsc.h"
*/
import "C"
import (
	"time"
)

var (
	// TscHz is TSC time units in one second.
	TscHz uint64
	// TscGHz is TSC time units in one nanosecond.
	TscGHz float64
	// TscSeconds is seconds in one TSC time unit.
	TscSeconds float64
	// TscNanos is nanoseconds in one TSC time unit.
	TscNanos float64
)

// InitTscUnit saves TSC time unit.
// This is called by ealinit package.
func InitTscUnit() {
	C.TscHz = C.rte_get_tsc_hz()
	TscHz = uint64(C.TscHz)
	TscGHz = float64(TscHz) / float64(time.Second)
	TscSeconds = 1 / float64(TscHz)
	TscNanos = 1 / TscGHz

	C.TscGHz = C.double(TscGHz)
	C.TscSeconds = C.double(TscSeconds)
	C.TscNanos = C.double(TscNanos)

	tsc1 := TscNow()
	unixRef := time.Now()
	tsc2 := TscNow()
	C.TscTimeRefUnixNano_ = C.double(unixRef.UnixNano())
	C.TscTimeRefTsc_ = (C.double(tsc1) + C.double(tsc2)) / 2.0
}

// TscTime represents a time point on TSC clock.
type TscTime uint64

// TscNow returns current TscTime.
func TscNow() TscTime {
	return TscTime(C.rte_get_tsc_cycles())
}

// Add returns t+d.
func (t TscTime) Add(d time.Duration) TscTime {
	return t + TscTime(ToTscDuration(d))
}

// Sub returns t-t0.
func (t TscTime) Sub(t0 TscTime) time.Duration {
	return FromTscDuration(int64(t - t0))
}

// ToTime converts to time.Time.
func (t TscTime) ToTime() time.Time {
	u := C.TscTime_ToUnixNano(C.TscTime(t))
	return time.Unix(0, int64(u))
}

// FromTscDuration converts TSC duration to time.Duration.
func FromTscDuration(d int64) time.Duration {
	return time.Duration(TscNanos * float64(d))
}

// ToTscDuration converts time.Duration to TSC duration.
func ToTscDuration(d time.Duration) int64 {
	return int64(float64(d) / TscNanos)
}
