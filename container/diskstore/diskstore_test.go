package diskstore_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/diskstore"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestDiskStore(t *testing.T) {
	defer ealthread.DefaultAllocator.Clear()
	assert, require := makeAR(t)

	device, e := bdev.NewMalloc(diskstore.BlockSize, 256)
	require.NoError(e)
	defer device.Close()

	th, e := spdkenv.NewThread()
	require.NoError(e)
	defer th.Close()
	require.NoError(ealthread.Launch(th))

	mp, e := pktmbuf.NewPool(ndni.PacketMempool.Config(), eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	store, e := diskstore.New(device, th, mp, 8)
	require.NoError(e)
	defer store.Close()

	minSlotID, maxSlotID := store.SlotRange()
	assert.Equal(uint64(1), minSlotID)
	assert.Equal(uint64(31), maxSlotID)

	dataLens := make([]int, 33)
	dataLens[2] = 1024

	for _, n := range []uint64{1, 31, 32} {
		data := makeData(fmt.Sprintf("/A/%d", n), time.Duration(n)*time.Millisecond)
		dataLens[n] = data.Mbuf().Len()
		store.PutData(n, data)
	}
	time.Sleep(100 * time.Millisecond) // give time for asynchronous PutData operation

	for _, n := range []uint64{1, 31} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, dataLens[n], interest)
		if !assert.NoError(e, n) {
			continue
		}
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.ToNPacket().Data.Freshness, n)
			data.Close()
		}
		interest.Close()
	}

	for _, n := range []uint64{2, 32} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, dataLens[n], interest)
		if !assert.NoError(e, n) {
			continue
		}
		assert.Nil(data, n)
		interest.Close()
	}

	assert.Zero(mp.CountInUse())
}
