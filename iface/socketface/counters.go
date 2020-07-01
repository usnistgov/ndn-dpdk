package socketface

import (
	"fmt"
)

// ExCounters contains extended counters.
type ExCounters struct {
	NRedials      int
	RxQueueLength int
	TxQueueLength int
}

func (cnt ExCounters) String() string {
	return fmt.Sprintf("%dredials, rx %dqueued, tx %dqueued", cnt.NRedials, cnt.RxQueueLength, cnt.RxQueueLength)
}

// ReadExCounters reads extended counters.
func (face *SocketFace) ReadExCounters() interface{} {
	return ExCounters{
		NRedials:      face.inner.NRedials,
		RxQueueLength: len(face.inner.Rx()),
		TxQueueLength: len(face.inner.Tx()),
	}
}
