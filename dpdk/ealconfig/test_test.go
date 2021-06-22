package ealconfig_test

import (
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

func init() {
	ealconfig.PmdPath = "/tmp/pmd-path"
}

var makeAR = testenv.MakeAR

func commaSetEquals(a *assert.Assertions, expected string, actual string, msgAndArgs ...interface{}) bool {
	expectedSet := strings.Split(expected, ",")
	actualSet := strings.Split(actual, ",")
	return a.ElementsMatch(expectedSet, actualSet, msgAndArgs...)
}
