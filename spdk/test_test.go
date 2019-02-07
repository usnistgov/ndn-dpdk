package spdk_test

import (
	"fmt"
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	eal := dpdktestenv.InitEal()

	if e := spdk.Init(eal, eal.Slaves[0]); e != nil {
		panic(fmt.Sprintf("spdk.InitEnv error %v", e))
	}

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
