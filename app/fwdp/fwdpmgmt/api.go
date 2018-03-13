package fwdpmgmt

import (
	"errors"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pit"
)

func Enable(dp *fwdp.DataPlane) {
	dim := &DpInfoMgmt{dp}
	appinit.MgmtRpcServer.RegisterName("DPInfo", dim)
}

type DpInfoMgmt struct {
	dp *fwdp.DataPlane
}

func (dim *DpInfoMgmt) Global(args struct{}, reply *FwdpInfo) error {
	reply.NInputs, reply.NFwds = dim.dp.CountLCores()
	return nil
}

func (dim *DpInfoMgmt) Input(arg IndexArg, reply *fwdp.InputInfo) error {
	reply1 := dim.dp.ReadInputInfo(arg.Index)
	if reply1 == nil {
		return errors.New("index out of range")
	}
	*reply = *reply1
	return nil
}

func (dim *DpInfoMgmt) Fwd(arg IndexArg, reply *fwdp.FwdInfo) error {
	reply1 := dim.dp.ReadFwdInfo(arg.Index)
	if reply1 == nil {
		return errors.New("index out of range")
	}
	*reply = *reply1
	return nil
}

func (dim *DpInfoMgmt) Pit(arg IndexArg, reply *pit.Counters) error {
	pcct := dim.dp.GetFwdPcct(arg.Index)
	if pcct == nil {
		return errors.New("index out of range")
	}
	pit := pit.Pit{pcct}
	*reply = pit.ReadCounters()
	return nil
}

func (dim *DpInfoMgmt) Cs(arg IndexArg, reply *CsCounters) error {
	pcct := dim.dp.GetFwdPcct(arg.Index)
	if pcct == nil {
		return errors.New("index out of range")
	}
	pit, cs := pit.Pit{pcct}, cs.Cs{pcct}
	pitCnt := pit.ReadCounters()

	reply.Capacity = cs.GetCapacity()
	reply.NEntries = cs.Len()
	reply.NHits = pitCnt.NCsMatch
	reply.NMisses = pitCnt.NInsert + pitCnt.NFound

	return nil
}
