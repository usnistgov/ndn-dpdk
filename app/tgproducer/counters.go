package tgproducer

/*
#include "../../csrc/tgproducer/producer.h"
*/
import "C"
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

// ReadCounters retrieves counters.
func (producer *Producer) ReadCounters() (cnt Counters) {
	for i := 0; i < int(producer.c.nPatterns); i++ {
		patternC := producer.c.pattern[i]
		var pcnt PatternCounters
		for j := 0; j < int(patternC.nReplies); j++ {
			replyC := patternC.reply[j]
			pcnt.PerReply = append(pcnt.PerReply, uint64(replyC.nInterests))
			pcnt.NInterests += uint64(replyC.nInterests)
		}
		cnt.PerPattern = append(cnt.PerPattern, pcnt)
		cnt.NInterests += pcnt.NInterests
	}
	cnt.NNoMatch = uint64(producer.c.nNoMatch)
	cnt.NAllocError = uint64(producer.c.nAllocError)
	return cnt
}
