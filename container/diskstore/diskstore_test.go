package diskstore_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/diskstore"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"go4.org/must"
)

func TestDiskStore(t *testing.T) {
	assert, require := makeAR(t)
	defer ealthread.AllocClear()

	device, e := bdev.NewMalloc(diskstore.BlockSize, 256)
	require.NoError(e)
	defer device.Close()

	th, e := spdkenv.NewThread()
	require.NoError(e)
	defer th.Close()
	require.NoError(ealthread.AllocLaunch(th))

	assert.Zero(packetPool.CountInUse())

	store, e := diskstore.New(device, th, 8)
	require.NoError(e)
	defer store.Close()

	minSlotID, maxSlotID := store.SlotRange()
	assert.Equal(uint64(1), minSlotID)
	assert.Equal(uint64(31), maxSlotID)

	dataLens := map[uint64]int{
		2: 1024,
	}
	for _, n := range []uint64{1, 31, 32} {
		data := makeData(fmt.Sprintf("/A/%d", n), time.Duration(n)*time.Millisecond)
		dataLens[n] = data.Mbuf().Len()
		store.PutData(n, data)
	}
	time.Sleep(100 * time.Millisecond) // give time for asynchronous PutData operation

	for _, n := range []uint64{1, 31} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		dataBuf := packetPool.MustAlloc(1)
		data, e := store.GetData(n, dataLens[n], interest, dataBuf[0])
		if !assert.NoError(e, n) {
			continue
		}
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.ToNPacket().Data.Freshness, n)
			must.Close(data)
		}
		must.Close(interest)
	}

	for _, n := range []uint64{2, 32} {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		dataBuf := packetPool.MustAlloc(1)
		data, e := store.GetData(n, dataLens[n], interest, dataBuf[0])
		if !assert.NoError(e, n) {
			continue
		}
		assert.Nil(data, n)
		must.Close(interest)
	}

	assert.Zero(packetPool.CountInUse())
}
