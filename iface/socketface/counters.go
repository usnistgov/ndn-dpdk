package socketface

import (
	"fmt"
)

// Extended counters.
type ExCounters struct {
	NRedials   int
	TxQueueCap int
	TxQueueLen int
}

func (cnt ExCounters) String() string {
	return fmt.Sprintf("%dredials, tx %dqueued %dmax", cnt.NRedials, cnt.TxQueueLen, cnt.TxQueueCap)
}

func (face *SocketFace) ReadExCounters() interface{} {
	return ExCounters{
		NRedials:   face.nRedials,
		TxQueueCap: cap(face.txQueue),
		TxQueueLen: len(face.txQueue),
	}
}
