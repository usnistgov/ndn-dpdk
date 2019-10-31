package pingserver

/*
#include "server.h"
*/
import "C"
import (
	"fmt"
	"strconv"
)

type PatternCounters struct {
	NInterests uint64
	PerReply   []uint64
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

type Counters struct {
	PerPattern  []PatternCounters
	NInterests  uint64
	NNoMatch    uint64
	NAllocError uint64
}

func (cnt Counters) String() string {
	s := fmt.Sprintf("%dI %dno-match %dalloc-error", cnt.NInterests, cnt.NNoMatch, cnt.NAllocError)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

func (server *Server) ReadCounters() (cnt Counters) {
	for i := 0; i < int(server.c.nPatterns); i++ {
		patternC := server.c.pattern[i]
		var pcnt PatternCounters
		for j := 0; j < int(patternC.nReplies); j++ {
			replyC := patternC.reply[j]
			pcnt.PerReply = append(pcnt.PerReply, uint64(replyC.nInterests))
			pcnt.NInterests += uint64(replyC.nInterests)
		}
		cnt.PerPattern = append(cnt.PerPattern, pcnt)
		cnt.NInterests += pcnt.NInterests
	}
	cnt.NNoMatch = uint64(server.c.nNoMatch)
	cnt.NAllocError = uint64(server.c.nAllocError)
	return cnt
}
