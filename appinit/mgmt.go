package appinit

import (
	"ndn-dpdk/mgmt"
)

func RegisterMgmt(mg interface{}) {
	e := mgmt.Register(mg)
	if e != nil {
		Exitf(EXIT_MGMT_ERROR, "RegisterMgmt(%T): %v", mg, e)
	}
}

func StartMgmt() {
	e := mgmt.Start()
	if e != nil {
		Exitf(EXIT_MGMT_ERROR, "StartMgmt: %v", e)
	}
}
