package bdev

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"go.uber.org/multierr"
)

// FileDriver indicates a file-backed block device driver.
type FileDriver string

// FileDriver values.
const (
	FileAio   FileDriver = "aio"
	FileUring FileDriver = "uring"
)

// File represents a file-backed block device.
// This may use either AIO or Uring driver.
type File struct {
	*Info
	driver   FileDriver
	filename string
}

var _ DeviceCloser = (*File)(nil)

// Filename returns the filename.
func (device *File) Filename() string {
	return device.filename
}

// Close destroys this block device.
// The underlying file is not deleted.
func (device *File) Close() error {
	return deleteByName(fmt.Sprintf("bdev_%s_delete", device.driver), device.Name())
}

// NewFileWithDriver opens a file-backed block device with specified driver.
func NewFileWithDriver(driver FileDriver, filename string, blockSize int) (device *File, e error) {
	initBdevLib()
	args := struct {
		Name      string `json:"name"`
		Filename  string `json:"filename"`
		BlockSize int    `json:"block_size,omitempty"`
	}{
		Name:      eal.AllocObjectID("bdev.File"),
		Filename:  filename,
		BlockSize: blockSize,
	}
	var name string
	if e = spdkenv.RPC(fmt.Sprintf("bdev_%s_create", driver), args, &name); e != nil {
		return nil, e
	}
	return &File{
		Info:     mustFind(name),
		driver:   driver,
		filename: filename,
	}, nil
}

// NewFile opens a file-backed block device.
func NewFile(filename string, blockSize int) (*File, error) {
	device, e0 := NewFileWithDriver(FileUring, filename, blockSize)
	if e0 == nil {
		return device, nil
	}
	device, e1 := NewFileWithDriver(FileAio, filename, blockSize)
	if e1 == nil {
		return device, nil
	}
	return nil, multierr.Append(e0, e1)
}
