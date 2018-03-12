package fwdpmgmt

import (
	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
)

func Enable(dp *fwdp.DataPlane) {
	dpm := &FwdpMgmt{dp}
	appinit.MgmtRpcServer.RegisterName("DataPlane", dpm)
}

type FwdpMgmt struct {
	dp *fwdp.DataPlane
}

func (dpm *FwdpMgmt) GetCounters(args struct{}, reply *fwdp.Counters) error {
	*reply = dpm.dp.ReadCounters()
	return nil
}
