package segmented_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"go4.org/must"
)

var (
	makeAR    = testenv.MakeAR
	randBytes = testenv.RandBytes
)

const fetchTimeout = 10 * time.Second

type ServeFetchFixture struct {
	t       testing.TB
	Bridge  *ndntestenv.Bridge
	Payload []byte
	SOpt    segmented.ServeOptions
	FOpt    segmented.FetchOptions
}

func (f *ServeFetchFixture) EnableBridge() {
	relay := ndntestenv.BridgeRelayConfig{
		Loss:     0.01,
		MinDelay: 40 * time.Millisecond,
		MaxDelay: 80 * time.Millisecond,
	}
	f.Bridge = ndntestenv.NewBridge(ndntestenv.BridgeConfig{
		RelayAB: relay,
		RelayBA: relay,
	})
	f.SOpt.Fw = f.Bridge.FwA
	f.FOpt.Fw = f.Bridge.FwB
	if f.FOpt.RetxLimit == 0 {
		f.FOpt.RetxLimit = 3
	}
}

func (f *ServeFetchFixture) Prepare(payloadLen, chunkSize int) {
	f.Payload = make([]byte, payloadLen)
	randBytes(f.Payload)
	f.SOpt.ChunkSize = chunkSize
}

func (f *ServeFetchFixture) Serve() (close func()) {
	_, require := makeAR(f.t)
	s, e := segmented.Serve(context.Background(), bytes.NewReader(f.Payload), f.SOpt)
	require.NoError(e)
	return func() { must.Close(s) }
}

func (f *ServeFetchFixture) Fetch() segmented.FetchResult {
	return segmented.Fetch(f.SOpt.Prefix, f.FOpt)
}

func NewServeFetchFixture(t testing.TB) (f *ServeFetchFixture) {
	f = &ServeFetchFixture{t: t}
	f.SOpt.Prefix = ndn.ParseName("/D")
	return f
}

func TestInexact(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewServeFetchFixture(t)
	fixture.EnableBridge()

	fixture.Prepare(10000, 3333)
	fixture.SOpt.DataSigner = ndn.DigestSigning
	fixture.FOpt.Verifier = ndn.DigestSigning
	defer fixture.Serve()()

	f := fixture.Fetch()
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	pkts, e := f.Packets(ctx)
	require.NoError(e)
	require.Len(pkts, 4)
	assert.Equal(len(pkts), f.Count())

	assert.Equal(fixture.Payload[0:3333], pkts[0].Content)
	assert.Equal(fixture.Payload[3333:6666], pkts[1].Content)
	assert.Equal(fixture.Payload[6666:9999], pkts[2].Content)
	assert.Equal(fixture.Payload[9999:10000], pkts[3].Content)

	assert.False(pkts[0].IsFinalBlock())
	assert.False(pkts[1].IsFinalBlock())
	assert.False(pkts[2].IsFinalBlock())
	assert.True(pkts[3].IsFinalBlock())
}

func TestExact(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewServeFetchFixture(t)
	fixture.EnableBridge()

	fixture.Prepare(4000, 2000)
	defer fixture.Serve()()

	f := fixture.Fetch()
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	chunks := make(chan []byte, 64)
	e := f.Chunks(ctx, chunks)
	require.NoError(e)
	require.Len(chunks, 2)

	chunk0 := <-chunks
	chunk1 := <-chunks
	assert.Equal(fixture.Payload[0:2000], chunk0)
	assert.Equal(fixture.Payload[2000:4000], chunk1)
}

func TestEmpty(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewServeFetchFixture(t)
	fixture.EnableBridge()

	fixture.Prepare(0, 1024)
	defer fixture.Serve()()

	f1 := fixture.Fetch()
	f2 := fixture.Fetch()
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	pkts, e := f1.Packets(ctx)
	require.NoError(e)
	require.Len(pkts, 1)
	assert.Len(pkts[0].Content, 0)
	assert.True(pkts[0].IsFinalBlock())

	payload, e := f2.Payload(ctx)
	require.NoError(e)
	require.Len(payload, 0)
}
