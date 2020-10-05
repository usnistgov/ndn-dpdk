package ealconfig_test

import (
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var (
	makeAR   = testenv.MakeAR
	fromJSON = testenv.FromJSON
	toJSON   = testenv.ToJSON
)

func commaSetEquals(a *assert.Assertions, expected string, actual string, msgAndArgs ...interface{}) bool {
	expectedSet := strings.Split(expected, ",")
	actualSet := strings.Split(actual, ",")
	return a.Subset(expectedSet, actualSet, msgAndArgs...) && a.Subset(actualSet, expectedSet, msgAndArgs...)
}
