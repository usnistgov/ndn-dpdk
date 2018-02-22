package dpdk

/*
#include <rte_config.h>
#include <rte_cycles.h>
*/
import "C"
import "time"

// TSC clock time point.
type TscTime uint64

// Get current TscTime.
func TscNow() TscTime {
	return TscTime(C.rte_get_tsc_cycles())
}

// Return t+d.
func (t TscTime) Add(d time.Duration) TscTime {
	return t + TscTime(convertToTscDuration(d))
}

// Return t1-t0.
func (t1 TscTime) Sub(t0 TscTime) time.Duration {
	return convertFromTscDuration(int64(t1 - t0))
}

// Convert to time.Time.
func (t TscTime) ToTime() time.Time {
	tsc1 := TscNow()
	std0 := time.Now()
	tsc2 := TscNow()

	tsc0 := TscTime((float64(tsc1) + float64(tsc2)) / 2.0)
	since := tsc0.Sub(t)
	return std0.Add(since)
}

func convertFromTscDuration(d int64) time.Duration {
	return time.Duration(float64(time.Second) / float64(C.rte_get_tsc_hz()) * float64(d))
}

func convertToTscDuration(d time.Duration) int64 {
	return int64(d.Seconds() * float64(C.rte_get_tsc_hz()))
}
