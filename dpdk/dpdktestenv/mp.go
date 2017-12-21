package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var DirectMp dpdk.PktmbufPool
var IndirectMp dpdk.PktmbufPool

func CreateDirectMp(capacity int, privSize uint16, dataRoomSize uint16) dpdk.PktmbufPool {
	createMp(&DirectMp, "TEST-MP-DIRECT", capacity, privSize, dataRoomSize)
	return DirectMp
}

func CreateIndirectMp(capacity int) dpdk.PktmbufPool {
	createMp(&IndirectMp, "TEST-MP-INDIRECT", capacity, 0, 0)
	return IndirectMp
}

func createMp(mp *dpdk.PktmbufPool, name string, capacity int, privSize uint16,
	dataRoomSize uint16) {
	InitEal()

	if mp.IsValid() {
		mp.Close()
	}

	var e error
	*mp, e = dpdk.NewPktmbufPool(name, capacity, 0, privSize, dataRoomSize, dpdk.NUMA_SOCKET_ANY)
	if e != nil {
		panic(fmt.Sprintf("dpdk.NewPktmbufPool(%s) error %v", name, e))
	}
}
