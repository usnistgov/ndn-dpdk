package bdev_test

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"go4.org/must"
)

const (
	blockSize  = 1024
	blockCount = 256
)

func checkSize(t testing.TB, device bdev.Device) {
	assert, _ := makeAR(t)

	bdi := device.DevInfo()
	assert.Equal(blockSize, bdi.BlockSize())
	assert.Equal(int64(blockCount), bdi.CountBlocks())
}

type bdevRWTest struct {
	headroom    mbuftestenv.Headroom
	blockOffset int64

	payload []byte
	sp      bdev.StoredPacket
}

func (rwt *bdevRWTest) Write(t testing.TB, bd *bdev.Bdev) {
	assert, _ := makeAR(t)

	segs := [][]byte{make([]byte, 500), make([]byte, 400), make([]byte, 134)}
	for _, seg := range segs {
		rand.Read(seg)
	}
	rwt.payload = bytes.Join(segs, nil)

	pkt := makePacket(rwt.headroom, segs[0], segs[1], segs[2])
	defer pkt.Close()

	var e error
	rwt.sp, e = bd.WritePacket(rwt.blockOffset, pkt)
	assert.NoError(e)
}

func (rwt *bdevRWTest) Read(t testing.TB, bd *bdev.Bdev) {
	assert, _ := makeAR(t)

	pkt := mbuftestenv.DirectMempool().MustAlloc(1)[0]
	defer pkt.Close()

	if assert.NoError(bd.ReadPacket(rwt.blockOffset, pkt, rwt.sp)) {
		assert.Equal(rwt.payload, pkt.Bytes())
	}
}

func makeRW2(device bdev.Device) (rwt0, rwt1 bdevRWTest) {
	rwt0.headroom = 0
	rwt1.headroom = 1

	nBlocks := device.DevInfo().CountBlocks()
	for rwt0.blockOffset == rwt1.blockOffset {
		rwt0.blockOffset, rwt0.blockOffset = rand.Int63n(nBlocks), rand.Int63n(nBlocks)
	}
	return
}

func doUnmap(t testing.TB, bd *bdev.Bdev) {
	assert, _ := makeAR(t)

	if bd.DevInfo().HasIOType(bdev.IOUnmap) {
		assert.NoError(bd.UnmapBlocks(0, 4))
	}
}

func testBdev(t testing.TB, device bdev.Device, mode bdev.Mode, ops ...func(t testing.TB, bd *bdev.Bdev)) {
	_, require := makeAR(t)

	bd, e := bdev.Open(device, mode)
	require.NoError(e)
	require.NotNil(bd)
	defer must.Close(bd)

	for _, op := range ops {
		op(t, bd)
	}
}

func TestMalloc(t *testing.T) {
	_, require := makeAR(t)

	device, e := bdev.NewMalloc(blockSize, blockCount)
	require.NoError(e)
	defer must.Close(device)

	checkSize(t, device)
	rwt0, rwt1 := makeRW2(device)
	testBdev(t, device, bdev.ReadWrite, rwt0.Write, rwt1.Write, rwt0.Read, rwt1.Read, doUnmap)
}

func TestDelayError(t *testing.T) {
	assert, require := makeAR(t)

	malloc, e := bdev.NewMalloc(blockSize, blockCount)
	require.NoError(e)
	defer must.Close(malloc)

	delay, e := bdev.NewDelay(malloc, bdev.DelayConfig{
		AvgReadLatency:  20 * time.Millisecond,
		P99ReadLatency:  30 * time.Millisecond,
		AvgWriteLatency: 40 * time.Millisecond,
		P99WriteLatency: 60 * time.Millisecond,
	})
	require.NoError(e)
	defer must.Close(delay)

	errInj, e := bdev.NewErrorInjection(delay)
	require.NoError(e)
	defer must.Close(errInj)

	checkSize(t, errInj)
	rwt0, rwt1 := makeRW2(errInj)
	testBdev(t, errInj, bdev.ReadWrite, rwt0.Write, rwt1.Write, rwt0.Read, rwt1.Read, doUnmap,
		func(t testing.TB, bd *bdev.Bdev) {
			assert.NoError(errInj.Inject(bdev.IORead, 2))
			pkt := mbuftestenv.DirectMempool().MustAlloc(1)[0]
			defer pkt.Close()
			e = bd.ReadPacket(rwt0.blockOffset, pkt, rwt0.sp)
			assert.Error(e)
		},
	)
}

func TestFile(t *testing.T) {
	assert, require := makeAR(t)
	filename := testenv.TempName(t)

	require.NoError(bdev.TruncateFile(filename, blockSize*blockCount))

	for _, ctor := range []func() (*bdev.File, error){
		func() (*bdev.File, error) { return bdev.NewFile(filename, blockSize) },
		func() (*bdev.File, error) { return bdev.NewFileWithDriver(bdev.FileAio, filename, blockSize) },
		func() (*bdev.File, error) { return bdev.NewFileWithDriver(bdev.FileUring, filename, blockSize) },
	} {
		device, e := ctor()
		require.NoError(e)
		assert.Equal(filename, device.Filename())
		checkSize(t, device)
		rwt0, rwt1 := makeRW2(device)
		testBdev(t, device, bdev.ReadWrite, rwt0.Write, rwt1.Write, rwt0.Read, rwt1.Read, doUnmap)
		must.Close(device)
	}
}

func TestNvme(t *testing.T) {
	assert, require := makeAR(t)

	envPCI, ok := os.LookupEnv("BDEVTEST_NVME")
	if !ok {
		t.Skip("NVMe test disabled; rerun test suite and specify device PCI address in BDEVTEST_NVME=00:00.0 environ.")
	}
	pciAddr, e := pciaddr.Parse(envPCI)
	require.NoError(e)

	nvme, e := bdev.AttachNvme(pciAddr)
	require.NoError(e)
	defer must.Close(nvme)

	require.Greater(len(nvme.Namespaces), 0)
	bdi := nvme.Namespaces[0]
	assert.True(bdi.HasIOType(bdev.IONvmeAdmin))
	assert.True(bdi.HasIOType(bdev.IONvmeIO))

	if os.Getenv("BDEVTEST_NVME_WRITE") == "1" {
		rwt0, rwt1 := makeRW2(bdi)
		testBdev(t, bdi, bdev.ReadWrite, rwt0.Write, rwt1.Write, rwt0.Read, rwt1.Read, doUnmap)
	} else {
		t.Log("NVMe write test disabled; rerun test suite with BDEVTEST_NVME_WRITE=1 environ to enable (will destroy data).")
		testBdev(t, bdi, bdev.ReadOnly)
	}
}
