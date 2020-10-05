package ealconfig_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

func TestDefaultHwInfoSource(t *testing.T) {
	printItem := func(obj interface{}) {
		j, _ := json.MarshalIndent(obj, "", "  ")
		fmt.Println(string(j))
	}
	if os.Getenv("EALCONFIG_SHOW_HWINFO") != "1" {
		fmt.Println("Set EALCONFIG_SHOW_HWINFO=1 to show DefaultHwInfoSource output")
		printItem = func(obj interface{}) {}
	}

	hwInfo := ealconfig.DefaultHwInfoSource()
	cores := hwInfo.Cores()
	printItem(cores)
}
