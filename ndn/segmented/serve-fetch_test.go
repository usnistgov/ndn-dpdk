package segmented_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

var makeAR = testenv.MakeAR

func TestSimple(t *testing.T) {
	assert, require := makeAR(t)

	payload := make([]byte, 10000)
	rand.Read(payload)

	var sOpt segmented.ServeOptions
	sOpt.Prefix = ndn.ParseName("/D")
	sOpt.DataSigner = ndn.DigestSigning
	sOpt.ChunkSize = 3333
	s, e := segmented.Serve(context.Background(), bytes.NewReader(payload), sOpt)
	require.NoError(e)
	defer s.Close()

	var fOpt segmented.FetchOptions
	fOpt.Verifier = ndn.DigestSigning
	f := segmented.Fetch(sOpt.Prefix, fOpt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pkts, e := f.Packets(ctx)
	require.NoError(e)
	require.Len(pkts, 4)

	assert.Equal(payload[0:3333], pkts[0].Content)
	assert.Equal(payload[3333:6666], pkts[1].Content)
	assert.Equal(payload[6666:9999], pkts[2].Content)
	assert.Equal(payload[9999:10000], pkts[3].Content)

	assert.False(pkts[0].IsFinalBlock())
	assert.False(pkts[1].IsFinalBlock())
	assert.False(pkts[2].IsFinalBlock())
	assert.True(pkts[3].IsFinalBlock())
}
