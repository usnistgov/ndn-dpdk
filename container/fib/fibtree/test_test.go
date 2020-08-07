package fibtree_test

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

var (
	makeAR    = testenv.MakeAR
	nameEqual = ndntestenv.NameEqual
	makeEntry = fibtestenv.MakeEntry
)
