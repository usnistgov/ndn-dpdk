package hrlog_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/hrlog"
	"github.com/usnistgov/ndn-dpdk/app/hrlog/hrlogreader"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestWriter(t *testing.T) {
	t.Cleanup(ealthread.AllocClear)
	assert, require := makeAR(t)
	filename := testenv.TempName(t)

	w, e := hrlog.NewWriter(hrlog.WriterConfig{
		Filename:     filename,
		Count:        1000,
		RingCapacity: 256,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))
	time.Sleep(100 * time.Millisecond)

	entries := make([]uint64, 8192)
	for i := range entries {
		entries[i] = rand.Uint64()
	}
	rs := urcu.NewReadSide()
	for i := 0; i < len(entries); i += 128 {
		hrlog.Post(rs, entries[i:i+128])
		time.Sleep(1 * time.Millisecond)
		rs.Quiescent()
	}
	rs.Close()

	time.Sleep(100 * time.Millisecond)
	w.Close()

	count := 0
	r, e := hrlogreader.Open(filename)
	require.NoError(e)
	for entry := range r.Read() {
		count++
		assert.Contains(entries, entry)
	}
	assert.Equal(1000, count)
}
