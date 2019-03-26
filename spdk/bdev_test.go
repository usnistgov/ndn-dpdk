package spdk_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/spdk"
)

const (
	bdevBlockSize  = 1024
	bdevBlockCount = 256
)

func testBdev(t *testing.T, bdi spdk.BdevInfo) {
	assert, require := makeAR(t)

	assert.Equal(bdevBlockSize, bdi.GetBlockSize())
	assert.Equal(bdevBlockCount, bdi.CountBlocks())

	bd, e := spdk.OpenBdev(bdi, spdk.BDEV_MODE_READ_WRITE)
	require.NoError(e)
	require.NotNil(bd)

	assert.Implements((*io.ReaderAt)(nil), bd)
	assert.Implements((*io.WriterAt)(nil), bd)

	_, e = bd.WriteAt([]byte{0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7}, 1021)
	assert.NoError(e)
	buf := make([]byte, 4)
	_, e = bd.ReadAt(buf, 1022)
	assert.NoError(e)
	assert.Equal([]byte{0xA1, 0xA2, 0xA3, 0xA4}, buf)

	pkt1 := dpdktestenv.PacketFromBytes(bytes.Repeat([]byte{0xB0}, 500), bytes.Repeat([]byte{0xB1}, 400), bytes.Repeat([]byte{0xB2}, 134))
	defer pkt1.Close()
	e = bd.WritePacket(100, 16, pkt1)
	assert.NoError(e)

	pkt2 := dpdktestenv.PacketFromBytes(bytes.Repeat([]byte{0xC0}, 124), bytes.Repeat([]byte{0xC1}, 400), bytes.Repeat([]byte{0xC2}, 510))
	defer pkt2.Close()
	e = bd.ReadPacket(100, 12, pkt2)
	assert.NoError(e)
	assert.Equal(pkt1.ReadAll(), pkt2.ReadAll())

	e = bd.Close()
	assert.NoError(e)
}

func TestMallocBdev(t *testing.T) {
	spdk.InitBdevLib()
	assert, require := makeAR(t)

	bdi, e := spdk.NewMallocBdev(bdevBlockSize, bdevBlockCount)
	require.NoError(e)

	testBdev(t, bdi)

	e = spdk.DestroyMallocBdev(bdi)
	assert.NoError(e)
}

func TestAioBdev(t *testing.T) {
	spdk.InitBdevLib()
	assert, require := makeAR(t)

	file, e := ioutil.TempFile("", "")
	require.NoError(e)
	require.NoError(file.Truncate(bdevBlockSize * bdevBlockCount))
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	bdi, e := spdk.NewAioBdev(filename, bdevBlockSize)
	require.NoError(e)

	testBdev(t, bdi)

	e = spdk.DestroyAioBdev(bdi)
	assert.NoError(e)
}

func TestNvmeBdev(t *testing.T) {
	spdk.InitBdevLib()
	assert, require := makeAR(t)

	nvmes, e := spdk.ListNvmes()
	require.NoError(e)
	if len(nvmes) == 0 {
		fmt.Println("skipping TestNvmeBdev: no NVMe drive available; rerun test suite with DPDKTESTENV_PCI=1 environ?")
		return
	}

	pciAddr := nvmes[0]
	fmt.Printf("%d NVMe drives available (%v), testing on %s\n", len(nvmes), nvmes, pciAddr)

	bdis, e := spdk.AttachNvmeBdevs(pciAddr)
	require.NoError(e)
	require.True(len(bdis) > 0)

	bdi := bdis[0]
	assert.True(bdi.IsNvme())
	// testBdev(t, bdi)

	e = spdk.DetachNvmeBdevs(pciAddr)
	assert.NoError(e)
}
