package ndtmgmt

import (
	"ndn-dpdk/ndn"
)

type UpdateArgs struct {
	Hash  uint64
	Name  *ndn.Name // If not empty, overrides Hash with the hash of this name.
	Value uint8
}

type UpdateReply struct {
	Index uint64
}
