package coretest

import (
	"testing"

	_ "github.com/usnistgov/ndn-dpdk/core" // ensure ndn-dpdk/core package compiles
)

func TestSipHash(t *testing.T) {
	testSipHash(t)
}
