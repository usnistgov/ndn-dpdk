package fib

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("fib")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
