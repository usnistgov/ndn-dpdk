package hrlog_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/hrlog"
	"github.com/usnistgov/ndn-dpdk/app/hrlog/hrlogreader"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestWriter(t *testing.T) {
	defer ealthread.AllocClear()
	assert, require := makeAR(t)

	filename, del := testenv.TempName()
	defer del()

	w, e := hrlog.NewWriter(hrlog.WriterConfig{
		RingCapacity: 256,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))

	task, e := w.Submit(hrlog.TaskConfig{
		Filename: filename,
		Count:    1024,
	})
	require.NoError(e)
	time.Sleep(50 * time.Millisecond)

	entries := make([]uint64, 8192)
	for i := range entries {
		entries[i] = rand.Uint64()
	}
	for i := 0; i < len(entries); i += 128 {
		hrlog.Post(entries[i : i+128])
		time.Sleep(1 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	assert.NoError(task.Stop())
	assert.NoError(w.Stop())

	count := 0
	r, e := hrlogreader.Open(filename)
	require.NoError(e)
	for entry := range r.Read() {
		count++
		assert.Contains(entries, entry)
	}
	assert.Equal(1024, count)
}
