package versionmgmt

import (
	"github.com/usnistgov/ndn-dpdk/mk/version"
)

type VersionMgmt struct{}

func (VersionMgmt) Version(args struct{}, reply *VersionReply) error {
	reply.Commit = version.Get().Commit
	return nil
}

type VersionReply struct {
	Commit string
}
