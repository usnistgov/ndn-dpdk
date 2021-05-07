package main

import "context"

type Version struct{}

func (Version) Version(args struct{}, reply *VersionReply) error {
	return client.Do(context.TODO(), `
		query version {
			version {
				Commit: commit
			}
		}
	`, nil, "version", reply)
}

type VersionReply struct {
	Commit string
}
