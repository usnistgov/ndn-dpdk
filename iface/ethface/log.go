package ethface

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/usnistgov/ndn-dpdk/core/logger"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
)

var (
	log           = logger.New("ethface")
	makeLogFields = logger.MakeFields
)

func newPortLogger(ethDev ethdev.EthDev) logrus.FieldLogger {
	return logger.NewWithPrefix("ethface", fmt.Sprintf("port %s", ethDev))
}
