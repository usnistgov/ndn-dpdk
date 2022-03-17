package main

type histogram struct {
	Act    uint8
	LCore  uint8
	Counts []int
}

func newHistogram(act uint8, lcore uint8) *histogram {
	return &histogram{
		Act:    act,
		LCore:  lcore,
		Counts: make([]int, 4096),
	}
}

func (h *histogram) Add(value int) {
	if value >= len(h.Counts) {
		nBins := len(h.Counts) * 2
		for value >= nBins {
			nBins *= 2
		}
		counts := make([]int, nBins)
		copy(counts, h.Counts)
		h.Counts = counts
	}
	h.Counts[value]++
}

func (h *histogram) Trim() {
	right := len(h.Counts) - 1
	for right > 0 && h.Counts[right] == 0 {
		right--
	}
	h.Counts = h.Counts[:right+1]
}
