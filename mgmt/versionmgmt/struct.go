package versionmgmt

import (
	"time"
)

type VersionReply struct {
	Commit    string
	BuildTime time.Time
}
