package versionmgmt

import (
	"ndn-dpdk/app/version"
)

type VersionMgmt struct{}

func (VersionMgmt) Version(args struct{}, reply *VersionReply) error {
	reply.Commit = version.COMMIT
	reply.BuildTime = version.GetBuildTime()
	return nil
}
