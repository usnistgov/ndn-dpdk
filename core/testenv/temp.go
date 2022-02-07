package testenv

import (
	"os"
	"path"
	"testing"
)

// TempDir creates a temporary directory.
// The temporary directory and contained files are automatically deleted during cleanup.
func TempDir(t testing.TB) (dir string) {
	dir, e := os.MkdirTemp("", "ndn-dpdk-test-*")
	if e != nil {
		panic(e)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TempName creates a temporary filename in a temporary directory.
// The temporary directory and contained files are automatically deleted during cleanup.
func TempName(t testing.TB, name ...string) (filename string) {
	dir := TempDir(t)
	switch len(name) {
	case 0:
		filename = "temp"
	default:
		filename = name[0]
	}
	return path.Join(dir, filename)
}
