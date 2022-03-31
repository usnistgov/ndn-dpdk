// Package version returns NDN-DPDK version information.
package version

import (
	"fmt"
	"runtime/debug"
	"time"
)

// Version records NDN-DPDK version information.
type Version struct {
	Version string    `json:"version"`
	Commit  string    `json:"commit"`
	Date    time.Time `json:"date"`
	Dirty   bool      `json:"dirty"`
}

func (v Version) String() string {
	return v.Version
}

// V contains NDN-DPDK version information.
var V = Version{
	Version: "development",
	Commit:  "unknown",
	Date:    time.Now(),
	Dirty:   true,
}

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	bs := map[string]string{}
	for _, kv := range bi.Settings {
		bs[kv.Key] = kv.Value
	}
	dt, e := time.Parse(time.RFC3339, bs["vcs.time"])
	if bs["vcs"] != "git" || len(bs["vcs.revision"]) != 40 || e != nil {
		return
	}

	V.Commit = bs["vcs.revision"]
	V.Date = dt
	V.Dirty = bs["vcs.modified"] == "true"
	V.Version = fmt.Sprintf("v0.0.0-%s-%s%s", V.Date.Format("20060102150405"), V.Commit[:12], map[bool]string{true: "-dirty"}[V.Dirty])
}
