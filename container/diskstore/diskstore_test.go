package diskstore_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/container/diskstore"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestenv"
	"ndn-dpdk/spdk"
)

func TestDiskStore(t *testing.T) {
	assert, require := makeAR(t)

	bdi, e := spdk.NewMallocBdev(diskstore.BLOCK_SIZE, 256)
	require.NoError(e)
	defer spdk.DestroyMallocBdev(bdi)

	mp, e := pktmbuf.NewPool("TestDiskStore", ndn.PacketMempool.GetConfig(), eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	store, e := diskstore.New(bdi, spdk.MainThread, mp, 8)
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

	for _, n := range []uint64{1, 31} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, dataLens[n], interest)
		if !assert.NoError(e, n) {
			continue
		}
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.GetFreshnessPeriod(), n)
			ndntestenv.ClosePacket(data)
		}
		ndntestenv.ClosePacket(interest)
	}

	for _, n := range []uint64{2, 32} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, dataLens[n], interest)
		if !assert.NoError(e, n) {
			continue
		}
		assert.Nil(data, n)
		ndntestenv.ClosePacket(interest)
	}

	assert.Zero(mp.CountInUse())
}
