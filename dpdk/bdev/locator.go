package bdev

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
)

// Locator describes where to create or attach a block device.
type Locator struct {
	// Malloc=true creates a simulated block device in hugepages memory.
	Malloc bool `json:"malloc,omitempty"`

	// File, if not empty, specifies a filename and creates a block device backed by this file.
	// The file is automatically created and truncated to the required size.
	File string `json:"file,omitempty"`
	// FileDriver customizes the SPDK driver for the file-backed block device.
	FileDriver *FileDriver `json:"fileDriver,omitempty"`

	// PCIAddr, if not nil, attaches an NVMe device.
	// The first NVMe namespace that has the expected block size and block count is used.
	PCIAddr *pciaddr.PCIAddress `json:"pciAddr,omitempty"`
}

// Create creates a block device.
// It ensures block size is as expected, and a minimum number of blocks are present.
func (loc Locator) Create(minBlocks int64) (Device, io.Closer, error) {
	type devCloser interface {
		Device
		io.Closer
	}
	retDevCloser := func(dev devCloser, e error) (Device, io.Closer, error) {
		return dev, dev, e
	}

	switch {
	case loc.Malloc:
		return retDevCloser(NewMalloc(minBlocks))

	case loc.File != "":
		loc.File = filepath.Clean(loc.File)
		if e := TruncateFile(loc.File, RequiredBlockSize*minBlocks); e != nil {
			return nil, nil, e
		}

		if loc.FileDriver == nil {
			return retDevCloser(NewFile(loc.File))
		}
		return retDevCloser(NewFileWithDriver(*loc.FileDriver, loc.File))

	case loc.PCIAddr != nil:
		nvme, e := AttachNvme(*loc.PCIAddr)
		if e != nil {
			return nil, nil, e
		}

		for _, nn := range nvme.Namespaces {
			if nn.BlockSize() == RequiredBlockSize && nn.CountBlocks() >= minBlocks {
				return nn, nvme, nil
			}
		}

		nvme.Close()
		return nil, nil, fmt.Errorf("no NVMe namespace has at least %d blocks of %d octets", minBlocks, RequiredBlockSize)

	default:
		return nil, nil, errors.New("invalid bdev locator")
	}
}
