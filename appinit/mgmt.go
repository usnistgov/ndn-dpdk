package appinit

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/mgmt"
)

func RegisterMgmt(mg interface{}) {
	logEntry := log.WithField("mg", fmt.Sprintf("%T", mg))
	e := mgmt.Register(mg)
	if e != nil {
		logEntry.WithError(e).Fatal("mgmt module register failed")
	}
	log.Debug("mgmt module registered")
}

func StartMgmt() {
	e := mgmt.Start()
	if e != nil {
		log.WithError(e).Fatal("mgmt start failed")
	}
	log.Debug("mgmt started")
}
