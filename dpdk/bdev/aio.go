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
	return deleteByName("bdev_aio_delete", device.Name())
}

// NewAio opens a file-backed block device.
func NewAio(filename string, blockSize int) (device *Aio, e error) {
	initBdevLib()
	args := struct {
		Name      string `json:"name"`
		Filename  string `json:"filename"`
		BlockSize int    `json:"block_size,omitempty"`
	}{
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
