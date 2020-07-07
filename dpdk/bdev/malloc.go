package bdev

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// Malloc represents a memory-backed block device.
type Malloc struct {
	*Info
}

// NewMalloc creates a memory-backed block device.
func NewMalloc(blockSize int, nBlocks int) (device *Malloc, e error) {
	initBdevLib()
	initAccelEngine() // Malloc bdev depends on accelerator engine
	var args bdevMallocCreateArgs
	args.BlockSize = blockSize
	args.NumBlocks = nBlocks
	var name string
	if e = spdkenv.RPC("bdev_malloc_create", args, &name); e != nil {
		return nil, e
	}
	return &Malloc{Find(name)}, nil
}

// Close destroys this block device.
func (device *Malloc) Close() error {
	var args bdevMallocDeleteArgs
	args.Name = device.Name()
	var ok bool
	return spdkenv.RPC("bdev_malloc_delete", args, &ok)
}

// DevInfo implements Device interface.
func (device *Malloc) DevInfo() *Info {
	return device.Info
}

type bdevMallocCreateArgs struct {
	BlockSize int `json:"block_size"`
	NumBlocks int `json:"num_blocks"`
}

type bdevMallocDeleteArgs struct {
	Name string `json:"name"`
}
