package fib

import (
	"ndn-dpdk/core/logger"
)

var (
	log           = logger.New("Fib")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)
