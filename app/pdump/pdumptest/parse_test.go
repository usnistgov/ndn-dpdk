package pdumptest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func nameWire(prefix string) []byte {
	name := ndn.ParseName(prefix)
	wire, _ := name.MarshalBinary()
	return wire
}

func TestExtractNameL3(t *testing.T) {
	assert, _ := makeAR(t)
	assert.True(bytes.Equal(extractName(ndn.MakeInterest("/I/1")), nameWire("/I/1")))
	assert.True(bytes.Equal(extractName(ndn.MakeData("/D/1")), nameWire("/D/1")))
	assert.True(bytes.Equal(extractName(ndn.MakeNack(an.NackNoRoute, ndn.MakeInterest("/N/1")).ToPacket()), nameWire("/N/1")))
}

func TestExtractNameFragment(t *testing.T) {
	assert, require := makeAR(t)

	fragmenter := ndn.NewLpFragmenter(1000)
	data := ndn.MakeData("/D"+strings.Repeat("/Z", 800), make([]byte, 1100))
	frags, _ := fragmenter.Fragment(data.ToPacket())
	require.Len(frags, 4)

	name0 := extractName(frags[0])
	assert.True(bytes.HasPrefix(name0, nameWire("/D"+strings.Repeat("/Z", 240))))
	assert.True(bytes.HasPrefix(nameWire("/D"+strings.Repeat("/Z", 333)), name0))

	assert.Len(extractName(frags[1]), 0)
	assert.Len(extractName(frags[2]), 0)
	assert.Len(extractName(frags[3]), 0)
}
