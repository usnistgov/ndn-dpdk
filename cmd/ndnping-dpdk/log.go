package main

import (
	"github.com/usnistgov/ndn-dpdk/core/logger"
)

var (
	log           = logger.New("ndnping")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
