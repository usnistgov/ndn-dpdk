package ethface

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"ndn-dpdk/core/logger"
	"ndn-dpdk/dpdk"
)

var (
	log           = logger.New("ethface")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)

func newPortLogger(ethDev dpdk.EthDev) logrus.FieldLogger {
	return logger.NewWithPrefix("ethface", fmt.Sprintf("port %d", ethDev))
}
