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
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
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

// Writer is a pdump writer thread.
type Writer struct {
	ealthread.Thread
	c     *C.PdumpWriter
	queue *ringbuffer.Ring
	mp    *pktmbuf.Pool

	wg     sync.WaitGroup
	faces  map[iface.ID]string // faceID => Locator
	hasSHB bool
}

var _ ealthread.ThreadWithRole = (*Writer)(nil)

func (w *Writer) startDumper() {
	w.wg.Add(1)
}

func (w *Writer) stopDumper() {
	w.wg.Done()
}

// AddFace records a face as NgInterface.
func (w *Writer) AddFace(face iface.Face) {
	id, loc := face.ID(), iface.LocatorString(face.Locator())
	if w.faces[id] == loc {
		return
	}
	w.faces[id] = loc

	shb, idb := ngMakeHeader(id, loc)
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

	logger.Info("sent pcapng block", zap.Uint32("type", blockType))
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Writer) ThreadRole() string {
	return Role
}

// Close releases resources.
func (w *Writer) Close() error {
	e := w.Stop()
	logger.Info("PdumpWriter stopped", zap.Uintptr("queue", uintptr(unsafe.Pointer(w.c.queue))))
	w.wg.Wait()
	logger.Info("PdumpWriter close", zap.Uintptr("queue", uintptr(unsafe.Pointer(w.c.queue))))

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

	return e
}

// NewWriter creates a pdump writer thread.
func NewWriter(cfg WriterConfig) (w *Writer, e error) {
	cfg.applyDefaults()
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	w = &Writer{
		c:     (*C.PdumpWriter)(eal.Zmalloc("PdumpWriter", C.sizeof_PdumpWriter, cfg.Socket)),
		mp:    pktmbuf.Direct.Get(cfg.Socket),
		faces: map[iface.ID]string{},
	}
	w.c.filename = C.CString(cfg.Filename)
	w.c.maxSize = C.size_t(cfg.MaxSize)
	for i := range w.c.intf {
		w.c.intf[i] = math.MaxUint32
	}

	w.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.PdumpWriter_Run), unsafe.Pointer(w.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&w.c.stop)),
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

	logger.Info("PdumpWriter open", zap.Uintptr("queue", uintptr(unsafe.Pointer(w.c.queue))))
	return w, nil
}

func ngMakeHeader(id iface.ID, loc string) (shb, idb []byte) {
	intf := pcapgo.DefaultNgInterface
	intf.Name = strconv.Itoa(int(id))
	intf.Description = loc
	intf.LinkType = layers.LinkTypeLinuxSLL
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
