package hrlog_test

import (
	"context"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/hrlog"
	"github.com/usnistgov/ndn-dpdk/app/hrlog/hrlogreader"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestWriter(t *testing.T) {
	t.Cleanup(ealthread.AllocClear)
	assert, require := makeAR(t)

	dir, del := testenv.TempDir()
	defer del()
	fileA, fileB, fileC := path.Join(dir, "A.bin"), path.Join(dir, "B.bin"), path.Join(dir, "C.bin")

	w, e := hrlog.NewWriter(hrlog.WriterConfig{
		RingCapacity: 256,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))
	defer w.Stop()

	ctxA, cancelA := context.WithCancel(context.TODO())
	ctxB, cancelB := context.WithCancel(context.TODO())
	ctxC, cancelC := context.WithCancel(context.TODO())
	resA := w.Submit(ctxA, hrlog.TaskConfig{
		Filename: fileA,
		Count:    1000,
	})
	resB := w.Submit(ctxB, hrlog.TaskConfig{
		Filename: fileB,
		Count:    16384,
	})
	resC := w.Submit(ctxC, hrlog.TaskConfig{
		Filename: fileC,
		Count:    16384,
	})
	time.Sleep(200 * time.Millisecond)

	cancelB()
	assert.ErrorIs(<-resB, context.Canceled)

	entries := make([]uint64, 8192)
	for i := range entries {
		entries[i] = rand.Uint64()
	}
	for i := 0; i < len(entries); i += 128 {
		hrlog.Post(entries[i : i+128])
		time.Sleep(1 * time.Millisecond)
	}

	time.Sleep(300 * time.Millisecond)
	cancelA()
	cancelC()
	assert.NoError(<-resA)
	assert.ErrorIs(<-resC, context.Canceled)
	assert.FileExists(fileA)
	assert.NoFileExists(fileB)
	assert.FileExists(fileC)

	count := 0
	r, e := hrlogreader.Open(fileA)
	require.NoError(e)
	for entry := range r.Read() {
		count++
		assert.Contains(entries, entry)
	}
	assert.Equal(1000, count)
}
