package ealthread

/*
#include "../../csrc/dpdk/thread.h"
*/
import "C"
import "unsafe"

// LoadStat contains statistics of a polling thread.
type LoadStat struct {
	EmptyPolls uint64 `json:"emptyPolls"`
	ValidPolls uint64 `json:"validPolls"`
}

// Sub computes the difference.
func (s LoadStat) Sub(prev LoadStat) (diff LoadStat) {
	diff.EmptyPolls = s.EmptyPolls - prev.EmptyPolls
	diff.ValidPolls = s.ValidPolls - prev.ValidPolls
	return diff
}

func LoadStatFromPtr(ptr unsafe.Pointer) (s LoadStat) {
	c := (*C.ThreadLoadStat)(ptr)
	s.EmptyPolls = uint64(c.nPolls[0])
	s.ValidPolls = uint64(c.nPolls[1])
	return s
}

// ThreadWithLoadStat is an object that tracks thread load statistics.
type ThreadWithLoadStat interface {
	Thread
	ThreadLoadStat() LoadStat
}
