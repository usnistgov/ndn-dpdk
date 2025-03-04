package upf_test

import (
	"encoding/hex"
	"net/netip"
	"slices"
	"testing"

	"github.com/bobg/seqs"
	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/wmnsk/go-pfcp/ie"
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

type createdPDR struct {
	PDRID uint16
	TEID  uint32
	IPv4  netip.Addr
}

func gatherCreatedPDRs(t testing.TB, rsp []*ie.IE) []createdPDR {
	assert, _ := makeAR(t)
	return slices.Collect(seqs.Map(upf.FindIE(ie.CreatedPDR).IterWithin(rsp), func(item *ie.IE) (res createdPDR) {
		var e error
		var ok bool

		res.PDRID, e = item.PDRID()
		assert.NoError(e)
		if fteid, e := item.FTEID(); assert.NoError(e) {
			res.TEID = fteid.TEID
			res.IPv4, ok = netip.AddrFromSlice(fteid.IPv4Address)
			assert.True(ok)
		}
		return
	}))
}
