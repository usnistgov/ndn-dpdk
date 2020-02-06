package versionmgmt

import (
	"ndn-dpdk/app/version"
)

type VersionMgmt struct{}

func (VersionMgmt) Version(args struct{}, reply *VersionReply) error {
	reply.Commit = version.COMMIT
	return nil
}

type VersionReply struct {
	Commit string
}
