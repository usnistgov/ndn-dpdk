package hwinfo_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var makeAR = testenv.MakeAR

func TestDefault(t *testing.T) {
	assert, _ := makeAR(t)
	printItem := func(obj any) {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(obj)
	}
	if os.Getenv("HWINFOTEST_SHOW") != "1" {
		t.Log("Set HWINFOTEST_SHOW=1 to show output")
		printItem = func(obj any) {}
	}

	cores := hwinfo.Default.Cores()
	assert.NotEmpty(cores)
	printItem(cores)
}
