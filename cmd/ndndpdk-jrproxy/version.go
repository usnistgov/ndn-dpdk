package main

type Version struct{}

func (Version) Version(args struct{}, reply *VersionReply) error {
	return client.Do(`
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
