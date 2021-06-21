package bdev

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// Aio represents a file-backed block device.
type Aio struct {
	*Info
}

// NewAio opens a file-backed block device.
func NewAio(filename string, blockSize int) (device *Aio, e error) {
	initBdevLib()
	var args bdevAioCreateArgs
	args.Name = eal.AllocObjectID("bdev.Aio")
	args.Filename = filename
	args.BlockSize = blockSize
	var name string
	if e = spdkenv.RPC("bdev_aio_create", args, &name); e != nil {
		return nil, e
	}
	return &Aio{Find(name)}, nil
}

// Close destroys this block device.
// The underlying file is not deleted.
func (device *Aio) Close() error {
	var args bdevAioDeleteArgs
	args.Name = device.Name()
	var ok bool
	return spdkenv.RPC("bdev_aio_delete", args, &ok)
}

// DevInfo implements Device interface.
func (device *Aio) DevInfo() *Info {
	return device.Info
}

type bdevAioCreateArgs struct {
	Name      string `json:"name"`
	Filename  string `json:"filename"`
	BlockSize int    `json:"block_size,omitempty"`
}

type bdevAioDeleteArgs struct {
	Name string `json:"name"`
}
