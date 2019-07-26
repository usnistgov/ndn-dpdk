package pingmgmt

import (
	"time"
)

type IndexArg struct {
	Index int
}

type ClientStartArgs struct {
	Index         int           // Task index
	Interval      time.Duration // Interest sending Interval
	ClearCounters bool          // whether to clear counters
}

type ClientStopArgs struct {
	Index   int           // Task index
	RxDelay time.Duration // sleep duration between stopping TX and stopping RX
}
