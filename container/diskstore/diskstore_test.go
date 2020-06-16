package diskstore_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/diskstore"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/spdk/bdev"
	"github.com/usnistgov/ndn-dpdk/spdk/spdkenv"
)

func TestDiskStore(t *testing.T) {
	assert, require := makeAR(t)

	device, e := bdev.NewMalloc(diskstore.BlockSize, 256)
	require.NoError(e)
	defer device.Close()

	mp, e := pktmbuf.NewPool("TestDiskStore", ndn.PacketMempool.GetConfig(), eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	store, e := diskstore.New(device, spdkenv.MainThread, mp, 8)
	require.NoError(e)
	defer store.Close()

	minSlotId, maxSlotId := store.GetSlotIdRange()
	assert.Equal(uint64(1), minSlotId)
	assert.Equal(uint64(31), maxSlotId)

	dataLens := make([]int, 33)
	dataLens[2] = 1024

	for _, n := range []uint64{1, 31, 32} {
		data := makeData(fmt.Sprintf("/A/%d", n), time.Duration(n)*time.Millisecond)
		dataLens[n] = data.GetPacket().AsMbuf().Len()
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
			assert.Equal(time.Duration(n)*time.Millisecond, data.GetFreshnessPeriod(), n)
			closePacket(data)
		}
		closePacket(interest)
	}

	for _, n := range []uint64{2, 32} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, dataLens[n], interest)
		if !assert.NoError(e, n) {
			continue
		}
		assert.Nil(data, n)
		closePacket(interest)
	}

	assert.Zero(mp.CountInUse())
}
