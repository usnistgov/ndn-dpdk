package socketface

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/usnistgov/ndn-dpdk/core/logger"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var (
	log           = logger.New("socketface")
	makeLogFields = logger.MakeFields
)

func newLogger(id iface.FaceId) logrus.FieldLogger {
	return logger.NewWithPrefix("socketface", fmt.Sprintf("face %d", id))
}
