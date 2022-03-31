package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestJoinDevArgs(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal("", eal.JoinDevArgs(nil))
	assert.Equal("", eal.JoinDevArgs(map[string]any{}))

	assert.Contains([]string{"a=-1,B=str", "B=str,a=-1"},
		eal.JoinDevArgs(map[string]any{
			"a": -1,
			"B": "str",
			"c": nil,
		}))

	assert.Equal("override", eal.JoinDevArgs(map[string]any{
		"":  "override",
		"a": -1,
		"B": "ignored",
	}))
}
