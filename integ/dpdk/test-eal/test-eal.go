package main

import (
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
  "github.com/stretchr/testify/assert"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()

	args, e := dpdk.EalInit([]string{"testprog", "-c1", "-n1", "--no-pci", "--", "X"})
	assert.NoError(t, e, "EalInit failed")
	assert.Equal(t, []string{"testprog", "X"}, args)
}