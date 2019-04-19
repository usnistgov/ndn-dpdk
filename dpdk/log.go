package dpdk

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("dpdk")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
