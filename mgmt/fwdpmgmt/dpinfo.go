package fwdpmgmt

import (
	"errors"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pit"
)

type DpInfoMgmt struct {
	Dp *fwdp.DataPlane
}

func (mg DpInfoMgmt) Global(args struct{}, reply *FwdpInfo) error {
	reply.NInputs, reply.NFwds = mg.Dp.CountLCores()
	return nil
}

func (mg DpInfoMgmt) Input(arg IndexArg, reply *fwdp.InputInfo) error {
	reply1 := mg.Dp.ReadInputInfo(arg.Index)
	if reply1 == nil {
		return errors.New("index out of range")
	}
	*reply = *reply1
	return nil
}

func (mg DpInfoMgmt) Fwd(arg IndexArg, reply *fwdp.FwdInfo) error {
	reply1 := mg.Dp.ReadFwdInfo(arg.Index)
	if reply1 == nil {
		return errors.New("index out of range")
	}
	*reply = *reply1
	return nil
}

func (mg DpInfoMgmt) Pit(arg IndexArg, reply *pit.Counters) error {
	pcct := mg.Dp.GetFwdPcct(arg.Index)
	if pcct == nil {
		return errors.New("index out of range")
	}
	pit := pit.Pit{pcct}
	*reply = pit.ReadCounters()
	return nil
}

func (mg DpInfoMgmt) Cs(arg IndexArg, reply *CsCounters) error {
	pcct := mg.Dp.GetFwdPcct(arg.Index)
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
