package fwdpmgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pit"
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
	pit := mg.Dp.GetFwdPit(arg.Index)
	if pit == nil {
		return errors.New("index out of range")
	}
	*reply = pit.ReadCounters()
	return nil
}

func readCslCnt(cs *cs.Cs, cslId cs.ListId) (cnt CsListCounters) {
	cnt.Count = cs.CountEntries(cslId)
	cnt.Capacity = cs.GetCapacity(cslId)
	return cnt
}

func (mg DpInfoMgmt) Cs(arg IndexArg, reply *CsCounters) error {
	thePit, theCs := mg.Dp.GetFwdPit(arg.Index), mg.Dp.GetFwdCs(arg.Index)
	if thePit == nil || theCs == nil {
		return errors.New("index out of range")
	}
	pitCnt := thePit.ReadCounters()

	reply.MD = readCslCnt(theCs, cs.CSL_MD)
	reply.MI = readCslCnt(theCs, cs.CSL_MI)
	reply.NHits = pitCnt.NCsMatch
	reply.NMisses = pitCnt.NInsert + pitCnt.NFound

	return nil
}

type IndexArg struct {
	Index int
}

type FwdpInfo struct {
	NInputs int
	NFwds   int
}

type CsListCounters struct {
	Count    int
	Capacity int
}

type CsCounters struct {
	MD CsListCounters // in-memory direct entries
	MI CsListCounters // in-memory indirect entries

	NHits   uint64
	NMisses uint64
}
