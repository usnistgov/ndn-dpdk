package fwdp

/*
#include "../../csrc/fwdp/disk.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"math"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

// DiskConfig contains disk service thread configuration.
type DiskConfig struct {
	// Locator describes where to create or attach a block device.
	bdev.Locator

	// Overprovision is the ratio of block device size divided by CS disk capacity.
	// Setting this above 1.00 can reduce disk full errors due to some slots still occupied by async I/O.
	// Default is 1.05.
	Overprovision float64 `json:"overprovision"`

	// Bdev specifies the block device.
	// If set, Locator and Overprovision are ignored.
	Bdev bdev.Device `json:"-"`

	// BdevCloser allows closing the block device.
	BdevCloser io.Closer `json:"-"`

	csDiskCapacity int
}

func (cfg *DiskConfig) createDevice(nBlocks int64) (bdev.Device, io.Closer, error) {
	if cfg.Bdev != nil {
		return cfg.Bdev, cfg.BdevCloser, nil
	}

	if !(cfg.Overprovision >= 1.0) {
		cfg.Overprovision = 1.05
	}
	nBlocks = int64(math.Ceil(float64(nBlocks) * cfg.Overprovision))

	return cfg.Locator.Create(nBlocks)
}

// Disk represents a disk helper thread.
type Disk struct {
	*spdkenv.Thread
	id int
	c  *C.FwDisk

	bdev       bdev.Device
	bdevCloser io.Closer
	store      *disk.Store
	allocs     map[int]*disk.Alloc
}

var (
	_ ealthread.ThreadWithRole     = (*Disk)(nil)
	_ ealthread.ThreadWithLoadStat = (*Disk)(nil)
	_ DispatchThread               = (*Disk)(nil)
)

// DispatchThreadID implements DispatchThread interface.
func (fwdisk *Disk) DispatchThreadID() int {
	return fwdisk.id
}

func (fwdisk *Disk) String() string {
	return fmt.Sprintf("disk%d", fwdisk.id)
}

// DemuxOf implements DispatchThread interface.
func (fwdisk *Disk) DemuxOf(t ndni.PktType) *iface.InputDemux {
	if t == ndni.PktInterest {
		return iface.InputDemuxFromPtr(unsafe.Pointer(&fwdisk.c.output))
	}
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
	if fwdisk.bdevCloser != nil {
		errs = append(errs, fwdisk.bdevCloser.Close())
		fwdisk.bdev, fwdisk.bdevCloser = nil, nil
	}
	if fwdisk.Thread != nil {
		errs = append(errs, fwdisk.Thread.Close())
		fwdisk.Thread = nil
	}
	eal.Free(fwdisk.c)
	return errors.Join(errs...)
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Disk) ThreadRole() string {
	return RoleDisk
}

// newDisk creates a disk service thread.
func newDisk(id int, lc eal.LCore, demuxPrep *demuxPreparer, cfg DiskConfig) (fwdisk *Disk, e error) {
	fwdisk = &Disk{
		id: id,
	}
	defer func(d *Disk) {
		if e != nil {
			must.Close(d)
		}
	}(fwdisk)

	if fwdisk.Thread, e = spdkenv.NewThread(); e != nil {
		return nil, e
	}

	calc := disk.SizeCalc{
		NThreads:   len(demuxPrep.Fwds),
		NPackets:   cfg.csDiskCapacity,
		PacketSize: ndni.PacketMempool.Config().Dataroom,
	}
	if fwdisk.bdev, fwdisk.bdevCloser, e = cfg.createDevice(calc.MinBlocks()); e != nil {
		return nil, e
	}

	socket := lc.NumaSocket()
	fwdisk.c = eal.ZmallocAligned[C.FwDisk]("FwDisk", C.sizeof_FwDisk, 1, socket)
	fwdisk.SetLCore(lc)

	if fwdisk.store, e = disk.NewStore(fwdisk.bdev, fwdisk.Thread, calc.BlocksPerSlot(),
		disk.StoreGetDataCallback.C(C.FwDisk_GotData, fwdisk.c)); e != nil {
		return nil, e
	}

	fwdisk.allocs = map[int]*disk.Alloc{}
	for i, fwd := range demuxPrep.Fwds {
		alloc := disk.NewAllocIn(fwdisk.store, i, len(demuxPrep.Fwds), fwd.NumaSocket())
		fwdisk.allocs[fwd.id] = alloc
		if e = fwd.Cs().SetDisk(fwdisk.store, alloc); e != nil {
			return nil, fmt.Errorf("Cs[%d].SetDisk: %w", fwd.id, e)
		}
	}

	demuxPrep.Prepare(fwdisk, socket)
	return fwdisk, nil
}
