package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var mps = make(map[string]dpdk.PktmbufPool)

func MakeMp(id string, capacity int, privSize uint16, dataRoomSize uint16) dpdk.PktmbufPool {
	InitEal()

	name := "TEST-MP-" + id
	if mp, ok := mps[name]; ok {
		mp.Close()
	}

	mp, e := dpdk.NewPktmbufPool(name, capacity, 0, privSize, dataRoomSize, dpdk.NUMA_SOCKET_ANY)
	if e != nil {
		panic(fmt.Sprintf("dpdk.NewPktmbufPool(%s) error %v", name, e))
	}

	mps[id] = mp
	return mp
}

func GetMp(id string) dpdk.PktmbufPool {
	if mp, ok := mps[id]; ok {
		return mp
	}

	panic(fmt.Sprintf("GetMp(%s) without MakeMp", id))
}

const MPID_DIRECT = "_default_direct"
const MPID_INDIRECT = "_default_indirect"

func MakeDirectMp(capacity int, privSize uint16, dataRoomSize uint16) dpdk.PktmbufPool {
	return MakeMp(MPID_DIRECT, capacity, privSize, dataRoomSize)
}

func MakeIndirectMp(capacity int) dpdk.PktmbufPool {
	return MakeMp(MPID_INDIRECT, capacity, 0, 0)
}
