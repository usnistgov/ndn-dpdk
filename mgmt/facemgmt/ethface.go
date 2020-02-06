package facemgmt

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/ethface"
)

type EthFaceMgmt struct{}

func (EthFaceMgmt) ListPorts(args struct{}, reply *[]PortInfo) error {
	result := make([]PortInfo, 0)
	for _, dev := range dpdk.ListEthDevs() {
		result = append(result, makePortInfo(dev))
	}
	*reply = result
	return nil
}

func (EthFaceMgmt) ListPortFaces(args PortArg, reply *[]BasicInfo) error {
	dev := dpdk.FindEthDev(args.Port)
	if !dev.IsValid() {
		return errors.New("EthDev not found")
	}

	result := make([]BasicInfo, 0)
	if port := ethface.FindPort(dev); port != nil {
		for _, face := range port.ListFaces() {
			result = append(result, makeBasicInfo(face))
		}
	}
	*reply = result
	return nil
}

func (EthFaceMgmt) ReadPortStats(args PortStatsArg, reply *dpdk.EthStats) error {
	dev := dpdk.FindEthDev(args.Port)
	if !dev.IsValid() {
		return errors.New("EthDev not found")
	}

	*reply = dev.GetStats()
	if args.Reset {
		dev.ResetStats()
	}
	return nil
}

type PortArg struct {
	Port string
}

type PortInfo struct {
	Name       string          // port name
	NumaSocket dpdk.NumaSocket // NUMA socket
	Active     bool            // whether port is active
	ImplName   string          // internal implementation name
}

func makePortInfo(dev dpdk.EthDev) (info PortInfo) {
	info.Name = dev.GetName()
	info.NumaSocket = dev.GetNumaSocket()
	port := ethface.FindPort(dev)
	if port != nil {
		info.Active = true
		info.ImplName = port.GetImplName()
	}
	return info
}

type PortStatsArg struct {
	PortArg
	Reset bool
}
