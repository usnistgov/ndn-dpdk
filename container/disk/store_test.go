package disk_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestStore(t *testing.T) {
	assert, require := makeAR(t)
	t.Cleanup(ealthread.AllocClear)

	device, e := bdev.NewMalloc(disk.BlockSize, 256)
	require.NoError(e)
	defer device.Close()

	th, e := spdkenv.NewThread()
	require.NoError(e)
	defer th.Close()
	require.NoError(ealthread.AllocLaunch(th))

	assert.Zero(packetPool.CountInUse())

	store, e := disk.NewStore(device, th, 8, disk.StoreGetDataGo)
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

	getData := func(n uint64) *ndni.Packet {
		interest := makeInterest(fmt.Sprintf("/A/%d", n))
		defer interest.Close()
		dataBuf := packetPool.MustAlloc(1)[0]
		dataBuf.Append(make([]byte, dataLens[n]))
		return store.GetData(n, interest, dataBuf)
	}

	for _, n := range []uint64{1, 31} {
		data := getData(n)
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.ToNPacket().Data.Freshness, n)
			data.Close()
		}
	}

	for _, n := range []uint64{2, 32} {
		data := getData(n)
		assert.Nil(data, n)
	}

	assert.Zero(packetPool.CountInUse())
}
