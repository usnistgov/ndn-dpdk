package socketface

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"ndn-dpdk/core/logger"
	"ndn-dpdk/iface"
)

var (
	log           = logger.New("socketface")
	makeLogFields = logger.MakeFields
	addressOf     = logger.AddressOf
)

func newLogger(id iface.FaceId) logrus.FieldLogger {
	return logger.NewWithPrefix("socketface", fmt.Sprintf("face %d", id))
}
