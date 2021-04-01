// Package bpf provides access to compiled eBPF ELF objects.
package bpf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvBpfPath declares an environment variable to explicitly specify a directory containing compiled eBPF ELF objects.
const EnvBpfPath = "NDNDPDK_BPF_PATH"

// Kind indicates the kind of compiled eBPF ELF object.
type Kind string

// Kind definitions.
const (
	Strategy Kind = "strategy"
	XDP      Kind = "xdp"
)

// Find determines the filesystem path of a compiled eBPF ELF object.
// It first constructs a filename from the kind and short name.
// It then searches the file in the following locations:
//  1. The path specified in NDNDPDK_BPF_PATH environ.
//  2. ../lib/bpf relative to the executable.
//     From /usr/local/sbin/ndndpdk-svc, this step looks for eBPF objects in /usr/local/lib/bpf.
//  3. build/lib/bpf in the source tree.
//     This is used in unit tests, and is skipped if the executable is installed under /usr.
func (kind Kind) Find(name string) (path string, e error) {
	filename := "ndndpdk-" + string(kind) + "-" + name + ".o"
	elfPaths := []string{}

	if env, ok := os.LookupEnv(EnvBpfPath); ok {
		elfPaths = append(elfPaths, filepath.Join(env, filename))
	}

	inUsr := false
	if exe, e := os.Executable(); e == nil {
		inUsr = strings.HasPrefix(exe, "/usr/")
		elfPaths = append(elfPaths, filepath.Join(filepath.Dir(exe), "../lib/bpf", filename))
	}

	if _, source, _, ok := runtime.Caller(0); !inUsr && ok {
		elfPaths = append(elfPaths, filepath.Join(filepath.Dir(source), "../build/lib/bpf", filename))
	}

	for _, elf := range elfPaths {
		if path, e := filepath.EvalSymlinks(elf); e == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("%s ELF '%s' not found in %s; try specifying eBPF path via %s environ",
		kind, name, strings.Join(elfPaths, ","), EnvBpfPath)
}
