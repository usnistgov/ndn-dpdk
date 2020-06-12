package eal

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("eal")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
