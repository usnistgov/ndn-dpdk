package main

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("ndnpktcopy")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
