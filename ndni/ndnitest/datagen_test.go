package ndnitest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestDataGen(t *testing.T) {
	assert, require := makeAR(t)

	gen := ndni.NewDataGen(ndnitestenv.Payload.Alloc(), ndn.ParseName("/suffix"), 3016*time.Millisecond, []byte{0xC0, 0xC1})
	defer gen.Close()

	dataPkt := gen.Encode(ndnitestenv.Data.Alloc(), ndnitestenv.Indirect.Alloc(), ndn.ParseName("/prefix"))
	require.NotNil(dataPkt)
	data := dataPkt.ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/prefix/suffix", data)
	assert.Equal(3016*time.Millisecond, data.Freshness)
	assert.Equal([]byte{0xC0, 0xC1}, data.Content)
}
