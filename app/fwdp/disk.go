package fwdp

/*
#include "../../csrc/fwdp/disk.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"path"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
)

// DiskMalloc in DiskConfig.Filename specifies a memory-backed bdev.
const DiskMalloc = "Malloc"

// DiskConfig contains disk helper thread configuration.
type DiskConfig struct {
	// Filename is the file used as disk caching block device.
	//
	// The file is truncated to the appropriate size required for disk caching.
	// If the file does not exist, it is created automatically.
	//
	// If this is set to DiskMalloc, the disk helper creates a memory-backed bdev as a simulated disk.
	Filename string `json:"filename"`

	// Bdev specifies the block device.
	// If set, Filename are ignored.
	Bdev bdev.Device `json:"-"`

	csDiskCapacity int
}

func (cfg *DiskConfig) createDevice(minBlocks int64) (bdev.Device, error) {
	if cfg.Bdev != nil {
		return cfg.Bdev, nil
	}

	if cfg.Filename == "" {
		return nil, errors.New("filename is missing")
	}
	if cfg.Filename == DiskMalloc {
		return bdev.NewMalloc(disk.BlockSize, minBlocks)
	}

	cfg.Filename = path.Clean(cfg.Filename)
	file, e := os.Create(cfg.Filename)
	if e != nil {
		return nil, fmt.Errorf("os.Create(%s) error: %w", cfg.Filename, e)
	}
	size := disk.BlockSize * int64(minBlocks)
	if e := file.Truncate(size); e != nil {
		return nil, fmt.Errorf("file.Truncate(%d) error: %w", size, e)
	}
	file.Chmod(0o600)
	file.Close()

	return bdev.NewFile(cfg.Filename, disk.BlockSize)
}

// Disk represents a disk helper thread.
type Disk struct {
	*spdkenv.Thread
	id int
	c  *C.FwDisk

	bdev   bdev.Device
	store  *disk.Store
	allocs map[int]*disk.Alloc
}

var (
	_ ealthread.ThreadWithRole     = (*Disk)(nil)
	_ ealthread.ThreadWithLoadStat = (*Disk)(nil)
)

// Init initializes the disk helper.
func (fwdisk *Disk) Init(lc eal.LCore, demuxPrep *demuxPreparer, cfg DiskConfig) (e error) {
	defer func() {
		if e == nil {
			return
		}
	}()

	if fwdisk.Thread, e = spdkenv.NewThread(); e != nil {
		return e
	}

	calc := disk.SizeCalc{
		NThreads:   len(demuxPrep.Fwds),
		NPackets:   cfg.csDiskCapacity,
		PacketSize: ndni.PacketMempool.Config().Dataroom,
	}
	if fwdisk.bdev, e = cfg.createDevice(calc.MinBlocks()); e != nil {
		return e
	}

	socket := lc.NumaSocket()
	fwdisk.c = (*C.FwDisk)(eal.ZmallocAligned("FwDisk", C.sizeof_FwDisk, 1, socket))
	fwdisk.SetLCore(lc)

	if fwdisk.store, e = disk.NewStore(fwdisk.bdev, fwdisk.Thread, calc.BlocksPerSlot(),
		disk.StoreGetDataCallback.C(C.FwDisk_GotData, fwdisk.c)); e != nil {
		return e
	}

	fwdisk.allocs = map[int]*disk.Alloc{}
	for i, fwd := range demuxPrep.Fwds {
		alloc := calc.CreateAlloc(i, fwd.NumaSocket())
		fwdisk.allocs[fwd.id] = alloc
		if e = fwd.Cs().SetDisk(fwdisk.store, alloc); e != nil {
			return fmt.Errorf("Cs[%d].SetDisk: %w", fwd.id, e)
		}
	}

	demuxPrep.PrepareDemuxI(fwdisk.id, iface.InputDemuxFromPtr(unsafe.Pointer(&fwdisk.c.output)))

	return nil
}

// Close stops and releases the thread.
func (fwdisk *Disk) Close() error {
	errs := []error{}
	for id, alloc := range fwdisk.allocs {
		errs = append(errs, alloc.Close())
		delete(fwdisk.allocs, id)
	}
	if fwdisk.store != nil {
		errs = append(errs, fwdisk.store.Close())
		fwdisk.store = nil
	}
	if fwdisk.Thread != nil {
		errs = append(errs, fwdisk.Thread.Close())
		fwdisk.Thread = nil
	}
	eal.Free(fwdisk.c)
	return multierr.Combine(errs...)
}

func (fwdisk *Disk) String() string {
	return fmt.Sprintf("disk%d", fwdisk.id)
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Disk) ThreadRole() string {
	return RoleDisk
}

func newDisk(id int) *Disk {
	return &Disk{id: id}
}
