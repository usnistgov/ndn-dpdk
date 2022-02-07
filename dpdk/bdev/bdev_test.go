package bdev_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"go4.org/must"
)

const (
	blockSize  = 1024
	blockCount = 256
)

func checkSize(t *testing.T, device bdev.Device) {
	assert, _ := makeAR(t)

	bdi := device.DevInfo()
	assert.Equal(blockSize, bdi.BlockSize())
	assert.Equal(blockCount, bdi.CountBlocks())
}

func doRW(t *testing.T, bd *bdev.Bdev) {
	assert, _ := makeAR(t)

	pkt1 := makePacket(bytes.Repeat([]byte{0xB0}, 500), bytes.Repeat([]byte{0xB1}, 400), bytes.Repeat([]byte{0xB2}, 134))
	defer pkt1.Close()
	assert.NoError(bd.WritePacket(100, 16, *pkt1))

	pkt2 := makePacket(bytes.Repeat([]byte{0xC0}, 124), bytes.Repeat([]byte{0xC1}, 400), bytes.Repeat([]byte{0xC2}, 510))
	defer pkt2.Close()
	if assert.NoError(bd.ReadPacket(100, 12, *pkt2)) {
		assert.Equal(pkt1.Bytes(), pkt2.Bytes())
	}

	if bd.DevInfo().HasIOType(bdev.IOUnmap) {
		assert.NoError(bd.UnmapBlocks(0, 4))
	}
}

func testBdev(t *testing.T, device bdev.Device, mode bdev.Mode, ops ...func(t *testing.T, bd *bdev.Bdev)) {
	assert, require := makeAR(t)

	bd, e := bdev.Open(device, mode)
	require.NoError(e)
	require.NotNil(bd)

	for _, op := range ops {
		op(t, bd)
	}

	e = bd.Close()
	assert.NoError(e)
}

func TestMalloc(t *testing.T) {
	_, require := makeAR(t)

	device, e := bdev.NewMalloc(blockSize, blockCount)
	require.NoError(e)
	defer must.Close(device)

	checkSize(t, device)
	testBdev(t, device, bdev.ReadWrite, doRW)
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
	testBdev(t, errInj, bdev.ReadWrite, doRW,
		func(t *testing.T, bd *bdev.Bdev) {
			assert.NoError(errInj.Inject(bdev.IORead, 2))
			pkt3 := makePacket(make([]byte, blockSize), make([]byte, blockSize))
			defer pkt3.Close()
			e = bd.ReadPacket(100, 2, *pkt3)
			assert.Error(e)
		},
	)
}

func TestAio(t *testing.T) {
	_, require := makeAR(t)
	filename := testenv.TempName(t)

	file, e := os.Create(filename)
	require.NoError(e)
	require.NoError(file.Truncate(blockSize * blockCount))
	file.Close()

	device, e := bdev.NewAio(filename, blockSize)
	require.NoError(e)
	defer must.Close(device)

	checkSize(t, device)
	testBdev(t, device, bdev.ReadWrite, doRW)
}

func TestNvme(t *testing.T) {
	assert, require := makeAR(t)

	nvmes, e := bdev.ListNvmes()
	require.NoError(e)
	if len(nvmes) == 0 {
		t.Skip("no NVMe drive available")
	}

	pciAddr := nvmes[0]
	t.Logf("%d NVMe drives available (%v), testing on %s\n", len(nvmes), nvmes, pciAddr)

	nvme, e := bdev.AttachNvme(pciAddr)
	require.NoError(e)
	defer must.Close(nvme)

	require.Greater(len(nvme.Namespaces), 0)
	bdi := nvme.Namespaces[0]
	assert.True(bdi.HasIOType(bdev.IONvmeAdmin))
	assert.True(bdi.HasIOType(bdev.IONvmeIO))

	if os.Getenv("BDEVTEST_NVME_WRITE") == "1" {
		testBdev(t, bdi, bdev.ReadWrite, doRW)
	} else {
		t.Log("NVMe write test disabled; rerun test suite with BDEVTEST_NVME_WRITE=1 environ to enable (will destroy data).")
		testBdev(t, bdi, bdev.ReadOnly)
	}
}
