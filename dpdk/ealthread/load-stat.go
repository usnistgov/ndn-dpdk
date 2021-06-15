package ealthread

/*
#include "../../csrc/dpdk/thread.h"
*/
import "C"
import "unsafe"

// LoadStat contains polling thread workload statistics.
type LoadStat struct {
	// EmptyPolls is number of polls that processed zero item.
	EmptyPolls uint64 `json:"emptyPolls"`

	// ValidPolls is number of polls that processed non-zero items.
	ValidPolls uint64 `json:"validPolls"`

	// Items is number of processed items.
	Items uint64 `json:"items"`

	// ItemsPerPoll is average number of processed items per valid poll.
	ItemsPerPoll float64 `json:"itemsPerPoll,omitempty"`
}

// Sub computes the difference.
func (s LoadStat) Sub(prev LoadStat) (diff LoadStat) {
	diff.EmptyPolls = s.EmptyPolls - prev.EmptyPolls
	diff.ValidPolls = s.ValidPolls - prev.ValidPolls
	diff.Items = s.Items - prev.Items
	if diff.ValidPolls != 0 {
		diff.ItemsPerPoll = float64(diff.Items) / float64(diff.ValidPolls)
	}
	return diff
}

// LoadStatFromPtr copies *C.ThreadLoadStat to LoadStat.
func LoadStatFromPtr(ptr unsafe.Pointer) (s LoadStat) {
	c := (*C.ThreadLoadStat)(ptr)
	s.EmptyPolls = uint64(c.nPolls[0])
	s.ValidPolls = uint64(c.nPolls[1])
	s.Items = uint64(c.items)
	// don't populate s.ItemsPerPoll, because Items and ValidPolls can possibly wraparound
	return s
}

// ThreadWithLoadStat is an object that tracks thread load statistics.
type ThreadWithLoadStat interface {
	Thread
	ThreadLoadStat() LoadStat
}
