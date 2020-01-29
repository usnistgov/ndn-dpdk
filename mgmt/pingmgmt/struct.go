package pingmgmt

import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/core/nnduration"
	"ndn-dpdk/ndn"
)

type IndexArg struct {
	Index int
}

type ClientStartArgs struct {
	Index         int                    // Task index
	Interval      nnduration.Nanoseconds // Interest sending Interval
	ClearCounters bool                   // whether to clear counters
}

type ClientStopArgs struct {
	Index   int                    // Task index
	RxDelay nnduration.Nanoseconds // sleep duration between stopping TX and stopping RX
}

type FetchBenchmarkArgs struct {
	Index    int // Task index
	Name     *ndn.Name
	Warmup   nnduration.Milliseconds
	Interval nnduration.Milliseconds
	Count    int
}

type FetchBenchmarkReply struct {
	Counters []fetch.Counters
	Goodput  float64
}
