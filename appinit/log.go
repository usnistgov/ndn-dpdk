package appinit

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("appinit")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
