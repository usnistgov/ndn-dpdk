package diskstore_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/container/diskstore"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
	"ndn-dpdk/spdk"
)

func TestDiskStore(t *testing.T) {
	assert, require := makeAR(t)

	bdi, e := spdk.NewMallocBdev(diskstore.BLOCK_SIZE, 256)
	require.NoError(e)
	defer spdk.DestroyMallocBdev(bdi)

	mp := dpdktestenv.MakeMp("TestDiskStore", 255, ndn.SizeofPacketPriv(), diskstore.SizeofDataroom(8))
	defer mp.Close()

	store, e := diskstore.New(bdi, spdk.MainThread, mp, 8)
	require.NoError(e)
	defer store.Close()

	minSlotId, maxSlotId := store.GetSlotIdRange()
	assert.Equal(uint64(1), minSlotId)
	assert.Equal(uint64(31), maxSlotId)

	for _, n := range []uint64{1, 31, 32} {
		data := ndntestutil.MakeData(fmt.Sprintf("/A/%d", n), time.Duration(n)*time.Millisecond)
		store.PutData(n, data)
	}

	for _, n := range []uint64{1, 31} {
		interest := ndntestutil.MakeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, interest)
		if !assert.NoError(e, n) {
			continue
		}
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.GetFreshnessPeriod(), n)
			ndntestutil.ClosePacket(data)
		}
		ndntestutil.ClosePacket(interest)
	}

	for _, n := range []uint64{2, 32} {
		interest := ndntestutil.MakeInterest(fmt.Sprintf("/A/%d", n))
		data, e := store.GetData(n, interest)
		if !assert.NoError(e, n) {
			continue
		}
		assert.Nil(data, n)
		ndntestutil.ClosePacket(interest)
	}

	assert.Zero(mp.CountInUse())
}
