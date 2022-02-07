package fwdp

/*
#include "../../csrc/fwdp/disk.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
)

// Disk represents a disk helper thread.
type Disk struct {
	*spdkenv.Thread
	id int
	c  *C.FwDisk

	bdev   *bdev.Malloc
	store  *disk.Store
	allocs map[int]*disk.Alloc
}

var (
	_ ealthread.ThreadWithRole     = (*Disk)(nil)
	_ ealthread.ThreadWithLoadStat = (*Disk)(nil)
)

// Init initializes the disk helper.
func (fwdisk *Disk) Init(lc eal.LCore, demuxPrep *demuxPreparer, pcctCfg pcct.Config) (e error) {
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
		NPackets:   pcctCfg.CsDiskCapacity,
		PacketSize: ndni.PacketMempool.Config().Dataroom,
	}
	if fwdisk.bdev, e = bdev.NewMalloc(disk.BlockSize, calc.MinBlocks()); e != nil {
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
