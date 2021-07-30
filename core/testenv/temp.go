package testenv

import (
	"os"
	"path"
)

// TempDir creates a temporary directory.
func TempDir() (dir string, del func()) {
	dir, e := os.MkdirTemp("", "ndn-dpdk-test-*")
	if e != nil {
		panic(e)
	}
	return dir, func() { os.RemoveAll(dir) }
}

// TempName creates a temporary filename.
func TempName(name ...string) (filename string, del func()) {
	dir, del := TempDir()
	switch len(name) {
	case 0:
		filename = "temp"
	default:
		filename = name[0]
	}
	return path.Join(dir, filename), del
}
