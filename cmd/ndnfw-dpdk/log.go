package main

import (
	"github.com/usnistgov/ndn-dpdk/core/logger"
)

var (
	log           = logger.New("ndnfw")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
