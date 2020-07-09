package bdev_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

const (
	bdevBlockSize  = 1024
	bdevBlockCount = 256
)

func checkSize(t *testing.T, device bdev.Device) {
	assert, _ := makeAR(t)

	bdi := device.DevInfo()
	assert.Equal(bdevBlockSize, bdi.BlockSize())
	assert.Equal(bdevBlockCount, bdi.CountBlocks())
}

func testBdev(t *testing.T, device bdev.Device, mode bdev.Mode) {
	assert, require := makeAR(t)

	bd, e := bdev.Open(device, mode)
	require.NoError(e)
	require.NotNil(bd)

	if mode == bdev.ReadWrite {
		pkt1 := mbuftestenv.MakePacket(bytes.Repeat([]byte{0xB0}, 500), bytes.Repeat([]byte{0xB1}, 400), bytes.Repeat([]byte{0xB2}, 134))
		defer pkt1.Close()
		e = bd.WritePacket(100, 16, *pkt1)
		assert.NoError(e)

		pkt2 := mbuftestenv.MakePacket(bytes.Repeat([]byte{0xC0}, 124), bytes.Repeat([]byte{0xC1}, 400), bytes.Repeat([]byte{0xC2}, 510))
		defer pkt2.Close()
		e = bd.ReadPacket(100, 12, *pkt2)
		assert.NoError(e)
		assert.Equal(pkt1.ReadAll(), pkt2.ReadAll())
	}

	e = bd.Close()
	assert.NoError(e)
}

func TestMalloc(t *testing.T) {
	assert, require := makeAR(t)

	device, e := bdev.NewMalloc(bdevBlockSize, bdevBlockCount)
	require.NoError(e)

	checkSize(t, device)
	testBdev(t, device, bdev.ReadWrite)

	e = device.Close()
	assert.NoError(e)
}

func TestAio(t *testing.T) {
	assert, require := makeAR(t)

	file, e := ioutil.TempFile("", "")
	require.NoError(e)
	require.NoError(file.Truncate(bdevBlockSize * bdevBlockCount))
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	device, e := bdev.NewAio(filename, bdevBlockSize)
	require.NoError(e)

	checkSize(t, device)
	testBdev(t, device, bdev.ReadWrite)

	e = device.Close()
	assert.NoError(e)
}

func TestNvme(t *testing.T) {
	assert, require := makeAR(t)

	nvmes, e := bdev.ListNvmes()
	require.NoError(e)
	if len(nvmes) == 0 {
		fmt.Printf("skipping TestNvme: no NVMe drive available; rerun test suite with %s=1 environ?\n", ealtestenv.EnvPci)
		return
	}

	pciAddr := nvmes[0]
	fmt.Printf("%d NVMe drives available (%v), testing on %s\n", len(nvmes), nvmes, pciAddr)

	nvme, e := bdev.AttachNvme(pciAddr)
	require.NoError(e)

	require.True(len(nvme.Namespaces) > 0)
	bdi := nvme.Namespaces[0]
	assert.True(bdi.IsNvme())

	mode := bdev.ReadOnly
	if os.Getenv("BDEVTEST_NVME_WRITE") == "1" {
		mode = bdev.ReadWrite
	} else {
		fmt.Println("NVMe write test disabled; rerun test suite with BDEVTEST_NVME_WRITE=1 environ to enable (will destroy data).")
	}

	testBdev(t, bdi, mode)

	e = nvme.Close()
	assert.NoError(e)
}
