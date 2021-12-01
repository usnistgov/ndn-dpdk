package ealthread

// LoadStat contains polling thread workload statistics.
type LoadStat struct {
	// EmptyPolls is number of polls that processed zero item.
	EmptyPolls uint64 `json:"emptyPolls" gqldesc:"Polls that processed zero item."`

	// ValidPolls is number of polls that processed non-zero items.
	ValidPolls uint64 `json:"validPolls" gqldesc:"Polls that processed non-zero items."`

	// Items is count of processed items.
	Items uint64 `json:"items" gqldesc:"Count of processed items."`

	// ItemsPerPoll is average count of processed items per valid poll.
	// This is only available from Sub() return value.
	ItemsPerPoll float64 `json:"itemsPerPoll,omitempty" gqldesc:"Average count of processed items per valid poll."`
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

// ThreadWithLoadStat is an object that tracks thread load statistics.
type ThreadWithLoadStat interface {
	Thread
	ThreadLoadStat() LoadStat
}
