package upf_test

import (
	"encoding/hex"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/wmnsk/go-pfcp/message"
)

var makeAR = testenv.MakeAR

func parsePFCP(wireHex string) message.Message {
	wire, e := hex.DecodeString(wireHex)
	if e != nil {
		panic(e)
	}
	msg, e := message.Parse(wire)
	if e != nil {
		panic(e)
	}
	return msg
}
