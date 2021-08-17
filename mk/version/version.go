// Package version records NDN-DPDK version information.
package version

import (
	"fmt"
	"strconv"
	"time"

	"github.com/graphql-go/graphql"
)

// Variables replaced via -ldflags -X.
var (
	commit string
	date   string
	dirty  string
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

// Get returns version information.
func Get() (v Version) {
	dt, e := strconv.ParseInt(date, 10, 64)
	if e != nil || len(commit) != 40 {
		v.Version = "development"
		v.Commit = "unknown"
		v.Date = time.Now()
		v.Dirty = true
		return
	}

	v.Commit = commit
	v.Date = time.Unix(dt, 0)
	v.Dirty = dirty != ""
	dirtySuffix := ""
	if v.Dirty {
		dirtySuffix = "-dirty"
	}
	v.Version = fmt.Sprintf("v0.0.0-%s-%s%s", v.Date.Format("20060102150405"), commit[:12], dirtySuffix)
	return
}

// GqlVersionType is the GraphQL type for Version struct.
var GqlVersionType *graphql.Object

func init() {
	GqlVersionType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Version",
		Fields: graphql.Fields{
			"version": &graphql.Field{
				Type: graphql.String,
			},
			"commit": &graphql.Field{
				Type: graphql.String,
			},
			"date": &graphql.Field{
				Type: graphql.DateTime,
			},
			"dirty": &graphql.Field{
				Type: graphql.Boolean,
			},
		},
	})
}
