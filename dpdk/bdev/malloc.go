package bdev

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// Malloc represents a memory-backed block device.
type Malloc struct {
	*Info
}

var _ DeviceCloser = (*Malloc)(nil)

// Close destroys this block device.
func (device *Malloc) Close() error {
	return deleteByName("bdev_malloc_delete", device.Name())
}

// NewMalloc creates a memory-backed block device.
func NewMalloc(blockSize int, nBlocks int) (device *Malloc, e error) {
	initBdevLib()
	initAccelEngine() // Malloc bdev depends on accelerator engine
	args := struct {
		BlockSize int `json:"block_size"`
		NumBlocks int `json:"num_blocks"`
	}{
		BlockSize: blockSize,
		NumBlocks: nBlocks,
	}
	var name string
	if e = spdkenv.RPC("bdev_malloc_create", args, &name); e != nil {
		return nil, e
	}
	return &Malloc{mustFind(name)}, nil
}
