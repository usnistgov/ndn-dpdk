package socketface

import (
	"fmt"
)

// Extended counters.
type ExCounters struct {
	NRedials      int
	RxQueueCap    int
	RxQueueLen    int
	RxCongestions int
	TxQueueCap    int
	TxQueueLen    int
}

func (cnt ExCounters) String() string {
	return fmt.Sprintf("%dredials, rx %dqueued %dmax %ddropped, tx %dqueued %dmax", cnt.NRedials,
		cnt.RxQueueLen, cnt.RxQueueCap, cnt.RxCongestions, cnt.TxQueueLen, cnt.TxQueueCap)
}

func (face *SocketFace) ReadExCounters() interface{} {
	return ExCounters{
		NRedials:      face.nRedials,
		RxQueueCap:    cap(face.rxQueue),
		RxQueueLen:    len(face.rxQueue),
		RxCongestions: face.rxCongestions,
		TxQueueCap:    cap(face.txQueue),
		TxQueueLen:    len(face.txQueue),
	}
}
