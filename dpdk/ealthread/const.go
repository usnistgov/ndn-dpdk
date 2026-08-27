package ealthread

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_DPDK_THREAD_ENUM_H -out=../../csrc/dpdk/thread-enum.h .

// ThreadSleep constants.
// These are effective only if NDNDPDK_MK_THREADSLEEP=1 is set during compilation.
// If a C thread had an empty poll, i.e. processed zero packets or work items during an iteration,
// it sleeps for a short duration to reduce CPU utilization.
//
// Initially, the sleep duration is SleepMin.
// Once every SleepAdjustEvery consecutive empty polls, the sleep duration is adjusted as:
//
//	d = MIN(SleepMax, d * SleepMultiply / SleepDivide + SleepAdd)
//
// After a valid poll, i.e. processed non-zero packets, the sleep duration is reset to SleepMin.
//
// All durations are in nanoseconds unit.
const (
	SleepMin         = 1
	SleepMax         = 100000
	SleepAdjustEvery = 1 << 10
	SleepMultiply    = 11
	SleepDivide      = 10
	SleepAdd         = 0

	_ = "enumgen::ThreadCtrl_Sleep:Sleep"
)
