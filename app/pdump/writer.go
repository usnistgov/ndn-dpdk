// Package pdump implements a packet dumper.
package pdump

/*
#include "../../csrc/pdump/writer.h"
*/
import "C"
import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/google/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Role is writer thread role name.
const Role = "PDUMP"

var logger = logging.New("pdump")

// WriterConfig contains writer configuration.
type WriterConfig struct {
	Filename     string
	MaxSize      int
	RingCapacity int
	Socket       eal.NumaSocket
}

func (cfg *WriterConfig) applyDefaults() {
	cfg.Filename = filepath.Clean(cfg.Filename)
	if cfg.MaxSize == 0 {
		cfg.MaxSize = DefaultFileSize
	}
	cfg.RingCapacity = ringbuffer.AlignCapacity(cfg.RingCapacity, 64, 65536)
	if cfg.Socket.IsAny() {
		cfg.Socket = eal.RandomSocket()
	}
}

func (cfg WriterConfig) validate() error {
	errs := []error{}
	if cfg.Filename == "" {
		errs = append(errs, errors.New("filename is missing"))
	}
	if cfg.MaxSize < MinFileSize {
		errs = append(errs, fmt.Errorf("file size is less than %d", MinFileSize))
	}
	return multierr.Combine(errs...)
}

// Writer is a packet dump writer thread.
type Writer struct {
	ealthread.ThreadWithCtrl
	filename string
	c        *C.PdumpWriter
	queue    *ringbuffer.Ring
	mp       *pktmbuf.Pool

	nSources int32
	intfs    map[int]pcapgo.NgInterface
	hasSHB   bool
}

var (
	_ ealthread.ThreadWithRole     = (*Writer)(nil)
	_ ealthread.ThreadWithLoadStat = (*Writer)(nil)
)

// startSource records a source starting.
// The writer cannot be closed until all sources have stopped.
func (w *Writer) startSource() {
	n := atomic.AddInt32(&w.nSources, 1)
	if n <= 0 {
		panic("attempting to startSource on stopped Writer")
	}
}

// stopSource records a source stopping.
// The writer can be closed after all sources have stopped.
func (w *Writer) stopSource() {
	n := atomic.AddInt32(&w.nSources, -1)
	if n < 0 {
		panic("w.nSources is negative")
	}
}

// defineIntf defines an NgInterface.
// intf.Name, intf.Description, and intf.LinkType should be set; other fields are ignored.
// Caller should hold sourcesLock.
func (w *Writer) defineIntf(id int, intf pcapgo.NgInterface) {
	if w.intfs[id] == intf {
		return
	}
	w.intfs[id] = intf

	shb, idb := ngMakeHeader(id, intf)
	if !w.hasSHB {
		w.putBlock(shb, NgTypeSHB, math.MaxUint16)
		w.hasSHB = true
	}
	w.putBlock(idb, NgTypeIDB, uint16(id))
}

func (w *Writer) putBlock(block []byte, blockType uint32, port uint16) {
	vec, e := w.mp.Alloc(1)
	for ; e != nil; vec, e = w.mp.Alloc(1) {
		logger.Warn("mempool full for pcapng block, retrying", zap.Uint32("type", blockType))
		time.Sleep(10 * time.Millisecond)
	}
	vec[0].SetType32(blockType)
	vec[0].SetPort(port)
	if e := vec[0].Append(block); e != nil {
		// SHB and IDB should fit in default dataroom of DIRECT mempool
		panic(e)
	}

	for w.queue.Enqueue(vec) != 1 {
		logger.Warn("queue full for pcapng block, retrying", zap.Uint32("type", blockType))
		time.Sleep(10 * time.Millisecond)
	}

	logger.Debug("sent pcapng block", zap.Uint32("type", blockType))
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Writer) ThreadRole() string {
	return Role
}

// Close releases resources.
func (w *Writer) Close() error {
	if !atomic.CompareAndSwapInt32(&w.nSources, 0, -65536) {
		return errors.New("cannot stop Writer with active sources")
	}

	e := w.Stop()
	logger.Info("Writer stopped",
		zap.Uintptr("queue", uintptr(unsafe.Pointer(w.c.queue))),
		zap.Error(e),
	)

	if w.c != nil {
		C.free(unsafe.Pointer(w.c.filename))
		eal.Free(w.c)
		w.c = nil
	}

	if w.queue != nil {
		for {
			vec := make(pktmbuf.Vector, WriterBurstSize)
			nDeq := w.queue.Dequeue(vec)
			if nDeq == 0 {
				break
			}
			vec[:nDeq].Close()
		}
		w.queue.Close()
		w.queue = nil
	}

	return nil
}

// NewWriter creates a pdump writer thread.
func NewWriter(cfg WriterConfig) (w *Writer, e error) {
	cfg.applyDefaults()
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	w = &Writer{
		filename: cfg.Filename,
		c:        (*C.PdumpWriter)(eal.Zmalloc("PdumpWriter", C.sizeof_PdumpWriter, cfg.Socket)),
		mp:       pktmbuf.Direct.Get(cfg.Socket),
		intfs:    map[int]pcapgo.NgInterface{},
	}
	w.c.filename = C.CString(cfg.Filename)
	w.c.maxSize = C.size_t(cfg.MaxSize)
	for i := range w.c.intf {
		w.c.intf[i] = math.MaxUint32
	}

	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(unsafe.Pointer(C.PdumpWriter_Run), w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	defer func() {
		if e != nil {
			w.Close()
		}
	}()

	w.queue, e = ringbuffer.New(cfg.RingCapacity, cfg.Socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return nil, e
	}
	w.c.queue = (*C.struct_rte_ring)(w.queue.Ptr())

	logger.Info("Writer open",
		zap.String("filename", cfg.Filename),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(w.c.queue))),
	)
	return w, nil
}

func ngMakeHeader(id int, info pcapgo.NgInterface) (shb, idb []byte) {
	intf := pcapgo.DefaultNgInterface
	intf.Name = info.Name
	intf.Description = info.Description
	intf.LinkType = info.LinkType
	intf.SnapLength = 262144

	wOpt := pcapgo.DefaultNgWriterOptions
	wOpt.SectionInfo.Application = "NDN-DPDK"

	var b bytes.Buffer
	w, _ := pcapgo.NewNgWriterInterface(&b, intf, wOpt)
	w.Flush()

	for b.Len() >= 12 {
		totalLength := binary.LittleEndian.Uint32(b.Bytes()[4:])
		block := make([]byte, totalLength)
		b.Read(block)
		blockType := binary.LittleEndian.Uint32(block[0:])
		switch blockType {
		case NgTypeSHB:
			shb = block
		case NgTypeIDB:
			idb = block
		}
	}
	if b.Len() != 0 {
		panic("NgWriter incomplete block")
	}
	if len(shb) == 0 {
		panic("NgWriter missing SHB")
	}
	if len(idb) == 0 {
		panic("NgWriter missing IDB")
	}
	return
}
