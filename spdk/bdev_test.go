package spdk_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"ndn-dpdk/spdk"
)

func testBdev(t *testing.T, bdi spdk.BdevInfo) {
	assert, require := makeAR(t)

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

	e = bd.Close()
	assert.NoError(e)
}

func TestMallocBdev(t *testing.T) {
	spdk.InitBdevLib()
	assert, require := makeAR(t)

	bdi, e := spdk.NewMallocBdev(1024, 256)
	require.NoError(e)
	assert.Equal(1024, bdi.GetBlockSize())
	assert.Equal(256, bdi.CountBlocks())

	testBdev(t, bdi)

	e = spdk.DestroyMallocBdev(bdi)
	assert.NoError(e)
}

func TestAioBdev(t *testing.T) {
	spdk.InitBdevLib()
	assert, require := makeAR(t)

	file, e := ioutil.TempFile("", "")
	require.NoError(e)
	require.NoError(file.Truncate(1024 * 256))
	filename := file.Name()
	file.Close()
	defer os.Remove(filename)

	bdi, e := spdk.NewAioBdev(filename, 1024)
	require.NoError(e)
	assert.Equal(1024, bdi.GetBlockSize())
	assert.Equal(256, bdi.CountBlocks())

	testBdev(t, bdi)

	e = spdk.DestroyAioBdev(bdi)
	assert.NoError(e)
}
