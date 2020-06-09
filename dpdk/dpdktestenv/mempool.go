package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var mps = make(map[string]dpdk.PktmbufPool)

func MakeMp(id string, capacity int, privSize int, dataroomSize int) dpdk.PktmbufPool {
	InitEal()

	name := "TEST-MP-" + id
	if mp, ok := mps[id]; ok {
		mp.Close()
	}

	mp, e := dpdk.NewPktmbufPool(name, capacity, privSize, dataroomSize, dpdk.NumaSocket{})
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

func Alloc(id string) dpdk.Mbuf {
	m, e := GetMp(id).Alloc()
	if e != nil {
		panic(fmt.Sprintf("mp[%s].Alloc() error %v", id, e))
	}
	return m
}

func AllocBulk(id string, mbufs interface{}) {
	e := GetMp(id).AllocBulk(mbufs)
	if e != nil {
		panic(fmt.Sprintf("mp[%s].AllocBulk() error %v", id, e))
	}
}

const MPID_DIRECT = "_default_direct"
const MPID_INDIRECT = "_default_indirect"

func MakeDirectMp(capacity int, privSize int, dataRoomSize int) dpdk.PktmbufPool {
	return MakeMp(MPID_DIRECT, capacity, privSize, dataRoomSize)
}

func MakeIndirectMp(capacity int) dpdk.PktmbufPool {
	return MakeMp(MPID_INDIRECT, capacity, 0, 0)
}
