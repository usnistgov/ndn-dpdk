package ndnitest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestInterestTemplate(t *testing.T) {
	assert, require := makeAR(t)

	var tpl ndni.InterestTemplate
	tpl.Init("/prefix", ndn.CanBePrefixFlag, ndn.MakeFHDelegation(10, "/FH"), 1895*time.Millisecond)

	interestPkt := tpl.Encode(ndnitestenv.Interest.Alloc(), ndn.ParseName("/suffix"), 0xABF0E278)
	require.NotNil(interestPkt)
	interest := interestPkt.ToNPacket().Interest
	require.NotNil(interest)
	nameEqual(assert, "/prefix/suffix", interest)
	assert.True(interest.CanBePrefix)
	assert.False(interest.MustBeFresh)
	assert.Len(interest.ForwardingHint, 1)
	assert.Equal(1895*time.Millisecond, interest.Lifetime)
	assert.Equal(ndn.NonceFromUint(0xABF0E278), interest.Nonce)
}
