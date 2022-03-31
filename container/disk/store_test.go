package disk_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type StoreFixture struct {
	t             testing.TB
	Thread        *spdkenv.Thread
	Device        bdev.Device
	deviceClosers []bdev.DeviceCloser
	Store         *disk.Store
}

func (f *StoreFixture) close() {
	if f.Store != nil {
		f.Store.Close()
	}
	for _, closer := range f.deviceClosers {
		closer.Close()
	}
	if f.Thread != nil {
		f.Thread.Close()
	}
	ealthread.AllocClear()
}

func (f *StoreFixture) AddDevice(device bdev.Device, e error) {
	_, require := makeAR(f.t)
	require.NoError(e)

	if closer, ok := device.(bdev.DeviceCloser); ok {
		f.deviceClosers = append(f.deviceClosers, closer)
	}
	f.Device = device
}

func (f *StoreFixture) MakeStore(nBlocksPerSlot int) {
	_, require := makeAR(f.t)
	require.NotNil(f.Device)
	var e error
	f.Store, e = disk.NewStore(f.Device, f.Thread, nBlocksPerSlot, disk.StoreGetDataGo)
	require.NoError(e)
}

func (f *StoreFixture) PutData(slotID uint64, dataName string, dataArgs ...interface{}) bdev.StoredPacket {
	_, require := makeAR(f.t)
	data := makeData(dataName, dataArgs...)
	sp, e := f.Store.PutData(slotID, data)
	require.NoError(e)
	return sp
}

func (f *StoreFixture) GetData(slotID uint64, sp bdev.StoredPacket, interestName string, interestArgs ...interface{}) (data *ndni.Packet) {
	interest := makeInterest(interestName, interestArgs...)
	defer interest.Close()
	dataBuf := packetPool.MustAlloc(1)[0]
	return f.Store.GetData(slotID, interest, dataBuf, sp)
}

func NewStoreFixture(t testing.TB) (f *StoreFixture) {
	_, require := makeAR(t)
	var e error

	f = &StoreFixture{
		t: t,
	}
	t.Cleanup(f.close)

	f.Thread, e = spdkenv.NewThread()
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(f.Thread))

	return f
}

func TestStore(t *testing.T) {
	assert, _ := makeAR(t)
	f := NewStoreFixture(t)
	f.AddDevice(bdev.NewMalloc(disk.BlockSize, 256))
	f.MakeStore(8)

	minSlotID, maxSlotID := f.Store.SlotRange()
	assert.Equal(uint64(1), minSlotID)
	assert.Equal(uint64(31), maxSlotID)

	assert.Zero(packetPool.CountInUse())

	dataSps := map[uint64]bdev.StoredPacket{}
	for _, n := range []uint64{1, 31, 32} {
		dataSps[n] = f.PutData(n, fmt.Sprintf("/A/%d", n), time.Duration(n)*time.Millisecond)
	}
	dataSps[2] = dataSps[31]
	time.Sleep(100 * time.Millisecond) // give time for asynchronous PutData operation

	for _, n := range []uint64{1, 31} {
		data := f.GetData(n, dataSps[n], fmt.Sprintf("/A/%d", n))
		if assert.NotNil(data, n) {
			assert.Equal(time.Duration(n)*time.Millisecond, data.ToNPacket().Data.Freshness, n)
			data.Close()
		}
	}

	for _, n := range []uint64{2, 32} {
		data := f.GetData(n, dataSps[n], fmt.Sprintf("/A/%d", n))
		assert.Nil(data, n)
	}

	assert.Zero(packetPool.CountInUse())

	cnt := f.Store.Counters()
	assert.EqualValues(3, cnt.NPutDataBegin)
	assert.EqualValues(2, cnt.NPutDataSuccess)
	assert.EqualValues(1, cnt.NPutDataFailure)
	assert.EqualValues(4, cnt.NGetDataBegin)
	assert.EqualValues(0, cnt.NGetDataReuse)
	assert.EqualValues(2, cnt.NGetDataSuccess)
	assert.EqualValues(2, cnt.NGetDataFailure)
}

func TestStoreQueue(t *testing.T) {
	assert, _ := makeAR(t)
	f := NewStoreFixture(t)
	f.AddDevice(bdev.NewMalloc(disk.BlockSize, 256))
	f.AddDevice(bdev.NewDelay(f.Device, bdev.DelayConfig{
		AvgReadLatency:  100 * time.Millisecond,
		P99ReadLatency:  200 * time.Millisecond,
		AvgWriteLatency: 100 * time.Millisecond,
		P99WriteLatency: 200 * time.Millisecond,
	}))
	f.MakeStore(8)

	sp1a := f.PutData(1, "/A/1", make([]byte, 1777))
	var wg sync.WaitGroup
	for k := 0; k < 2; k++ {
		wg.Add(8)
		for i := 0; i < 4; i++ {
			go func(k, i int) {
				defer wg.Done()
				data := f.GetData(1, sp1a, "/A/1")
				if assert.NotNil(data, "%d %d", k, i) {
					data.Close()
				}
			}(k, i)
			go func(k, i int) {
				defer wg.Done()
				data := f.GetData(7, sp1a, "/A/1")
				assert.Nil(data, "%d %d", k, i)
			}(k, i)
		}
		wg.Wait()
	}

	cnt := f.Store.Counters()
	assert.EqualValues(1, cnt.NPutDataBegin)
	assert.EqualValues(1, cnt.NPutDataSuccess)
	assert.EqualValues(0, cnt.NPutDataFailure)
	assert.EqualValues(16, cnt.NGetDataBegin+cnt.NGetDataReuse)
	assert.GreaterOrEqual(cnt.NGetDataReuse, uint64(4))
	assert.EqualValues(8, cnt.NGetDataSuccess)
	assert.EqualValues(8, cnt.NGetDataFailure)
}
