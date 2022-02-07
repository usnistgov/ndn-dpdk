package bdev

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// Aio represents a file-backed block device.
type Aio struct {
	*Info
}

var _ Device = (*Aio)(nil)

// Close destroys this block device.
// The underlying file is not deleted.
func (device *Aio) Close() error {
	args := aioDeleteArgs{
		Name: device.Name(),
	}
	var ok bool
	return spdkenv.RPC("bdev_aio_delete", args, &ok)
}

// DevInfo implements Device interface.
func (device *Aio) DevInfo() *Info {
	return device.Info
}

// NewAio opens a file-backed block device.
func NewAio(filename string, blockSize int) (device *Aio, e error) {
	initBdevLib()
	args := aioCreateArgs{
		Name:      eal.AllocObjectID("bdev.Aio"),
		Filename:  filename,
		BlockSize: blockSize,
	}
	var name string
	if e = spdkenv.RPC("bdev_aio_create", args, &name); e != nil {
		return nil, e
	}
	return &Aio{Find(name)}, nil
}

type aioCreateArgs struct {
	Name      string `json:"name"`
	Filename  string `json:"filename"`
	BlockSize int    `json:"block_size,omitempty"`
}

type aioDeleteArgs struct {
	Name string `json:"name"`
}
