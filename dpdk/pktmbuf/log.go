package pktmbuf

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("pktmbuf")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
