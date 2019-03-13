package main

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("ndnping")
	tblog         = logger.New("ThroughputBenchmark")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
