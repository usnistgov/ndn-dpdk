package fwdp

import (
	"fmt"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

// LCoreAlloc roles.
const (
	LCoreRole_Input  = iface.LCoreRole_RxLoop
	LCoreRole_Output = iface.LCoreRole_TxLoop
	LCoreRole_Crypto = "CRYPTO"
	LCoreRole_Fwd    = "FWD"
)

// LCore allocator for dataplane.
type DpLCores struct {
	Allocator *dpdk.LCoreAllocator

	Inputs  []dpdk.LCore
	Outputs []dpdk.LCore
	Crypto  dpdk.LCore
	Fwds    []dpdk.LCore
}

// Allocate LCores for all necessary roles.
func (la *DpLCores) Alloc() (e error) {
	rxlTxlNuma := createface.ListRxTxNumaSockets()
	if la.Inputs, e = la.allocNuma(LCoreRole_Input, rxlTxlNuma); e != nil {
		return e
	}
	if la.Outputs, e = la.allocNuma(LCoreRole_Output, rxlTxlNuma); e != nil {
		return e
	}

	la.Crypto = la.Allocator.Alloc(LCoreRole_Crypto, dpdk.NumaSocket{})

	if la.Fwds = la.allocMax(LCoreRole_Fwd); len(la.Fwds) == 0 {
		return fmt.Errorf("no lcore available for %s", LCoreRole_Fwd)
	}

	return nil
}

// Allocate LCores on list of NumaSockets.
func (la *DpLCores) allocNuma(role string, numaSockets []dpdk.NumaSocket) (list []dpdk.LCore, e error) {
	for _, numaSocket := range numaSockets {
		if lc := la.Allocator.Alloc(role, numaSocket); lc.IsValid() {
			list = append(list, lc)
		} else {
			return nil, fmt.Errorf("no lcore available for %s", role)
		}
	}
	return list, nil
}

// Allocate all remaining LCores to a role.
func (la *DpLCores) allocMax(role string) (list []dpdk.LCore) {
	for {
		if lc := la.Allocator.Alloc(role, dpdk.NumaSocket{}); lc.IsValid() {
			list = append(list, lc)
		} else {
			break
		}
	}
	return list
}

// Release all allocated LCores.
func (la *DpLCores) Close() error {
	la.Allocator.Clear()
	return nil
}
