package eal

/*
#include "../../csrc/dpdk/tsc.h"
*/
import "C"
import (
	"time"
)

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
	tsc1 := TscNow()
	std0 := time.Now()
	tsc2 := TscNow()

	tsc0 := TscTime((float64(tsc1) + float64(tsc2)) / 2.0)
	since := tsc0.Sub(t)
	return std0.Add(since)
}

// GetNanosInTscUnit returns number of nanoseconds in a TSC time unit.
func GetNanosInTscUnit() float64 {
	return float64(time.Second) / float64(C.rte_get_tsc_hz())
}

// GetTscUnit returns TSC time unit as time.Duration.
func GetTscUnit() time.Duration {
	return time.Duration(GetNanosInTscUnit())
}

// FromTscDuration converts TSC duration to time.Duration.
func FromTscDuration(d int64) time.Duration {
	return time.Duration(GetNanosInTscUnit() * float64(d))
}

// ToTscDuration converts time.Duration to TSC duration.
func ToTscDuration(d time.Duration) int64 {
	return int64(float64(d) / GetNanosInTscUnit())
}
