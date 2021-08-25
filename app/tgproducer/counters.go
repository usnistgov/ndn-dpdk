package tgproducer

import (
	"fmt"
	"strconv"
)

// PatternCounters contains per-pattern counters.
type PatternCounters struct {
	NInterests uint64   `json:"nInterests"`
	PerReply   []uint64 `json:"perReply"`
}

func (cnt PatternCounters) String() string {
	var b []byte
	for i, n := range cnt.PerReply {
		if i > 0 {
			b = append(b, '+')
		}
		b = strconv.AppendUint(b, n, 10)
	}
	b = append(b, '=')
	b = strconv.AppendUint(b, cnt.NInterests, 10)
	b = append(b, 'I')
	return string(b)
}

// Counters contains producer counters.
type Counters struct {
	PerPattern  []PatternCounters `json:"perPattern"`
	NInterests  uint64            `json:"nInterests"`
	NNoMatch    uint64            `json:"nNoMatch"`
	NAllocError uint64            `json:"nAllocError"`
}

func (cnt Counters) String() string {
	s := fmt.Sprintf("%dI %dno-match %dalloc-error", cnt.NInterests, cnt.NNoMatch, cnt.NAllocError)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

func (w *worker) accumulateCounters(cnt *Counters) {
	for i := range cnt.PerPattern {
		patternC, pcnt := w.c.pattern[i], &cnt.PerPattern[i]
		for j := range pcnt.PerReply {
			replyC := patternC.reply[j]
			pcnt.PerReply[j] += uint64(replyC.nInterests)
			pcnt.NInterests += uint64(replyC.nInterests)
		}
		cnt.NInterests += pcnt.NInterests
	}
	cnt.NNoMatch += uint64(w.c.nNoMatch)
	cnt.NAllocError += uint64(w.c.nAllocError)
}

// Counters retrieves counters.
func (p Producer) Counters() (cnt Counters) {
	cnt.PerPattern = make([]PatternCounters, len(p.cfg.Patterns))
	for i, pattern := range p.cfg.Patterns {
		cnt.PerPattern[i].PerReply = make([]uint64, len(pattern.Replies))
	}

	for _, w := range p.workers {
		w.accumulateCounters(&cnt)
	}
	return cnt
}
