package fwdp

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
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
	Allocator *eal.LCoreAllocator

	Inputs  []eal.LCore
	Outputs []eal.LCore
	Crypto  eal.LCore
	Fwds    []eal.LCore
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

	la.Crypto = la.Allocator.Alloc(LCoreRole_Crypto, eal.NumaSocket{})

	if la.Fwds = la.allocMax(LCoreRole_Fwd); len(la.Fwds) == 0 {
		return fmt.Errorf("no lcore available for %s", LCoreRole_Fwd)
	}

	return nil
}

// Allocate LCores on list of NumaSockets.
func (la *DpLCores) allocNuma(role string, numaSockets []eal.NumaSocket) (list []eal.LCore, e error) {
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
func (la *DpLCores) allocMax(role string) (list []eal.LCore) {
	for {
		if lc := la.Allocator.Alloc(role, eal.NumaSocket{}); lc.IsValid() {
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
