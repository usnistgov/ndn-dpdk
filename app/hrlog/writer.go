package hrlog

/*
#include "../../csrc/hrlog/writer.h"
*/
import "C"
import (
	"errors"
	"path/filepath"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"go.uber.org/zap"
)

// Role is writer thread role name.
const Role = "HRLOG"

// TheWriter is the current Writer instance.
var TheWriter *Writer

// Error conditions.
var (
	ErrDisabled  = errors.New("hrlog module disabled")
	ErrQueueFull = errors.New("too many pending tasks")
	ErrWriter    = errors.New("writer failed")
)

// WriterConfig contains writer configuration.
type WriterConfig struct {
	Filename     string
	Count        int
	RingCapacity int
	Socket       eal.NumaSocket
}

func (cfg *WriterConfig) applyDefaults() {
	cfg.Filename = filepath.Clean(cfg.Filename)
	if cfg.Count == 0 {
		cfg.Count = 1 << 28 // 268 million samples, 2GB file
	}
	cfg.RingCapacity = ringbuffer.AlignCapacity(cfg.RingCapacity, 64, 65536)
	if cfg.Socket.IsAny() {
		cfg.Socket = eal.RandomSocket()
	}
}
func (cfg WriterConfig) validate() error {
	if cfg.Filename == "" {
		return errors.New("filename is missing")
	}
	return nil
}

// Writer is a hrlog writer thread.
type Writer struct {
	ealthread.ThreadWithCtrl
	filename string
	c        *C.HrlogWriter
	queue    *ringbuffer.Ring
}

var (
	_ ealthread.ThreadWithRole     = (*Writer)(nil)
	_ ealthread.ThreadWithLoadStat = (*Writer)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Writer) ThreadRole() string {
	return Role
}

// Close releases resources.
func (w *Writer) Close() error {
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
		w.queue.Close()
		w.queue = nil
	}

	return nil
}

// NewWriter creates a hrlog writer thread.
func NewWriter(cfg WriterConfig) (w *Writer, e error) {
	cfg.applyDefaults()
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	w = &Writer{
		filename: cfg.Filename,
		c:        eal.Zmalloc[C.HrlogWriter]("HrlogWriter", C.sizeof_HrlogWriter, cfg.Socket),
	}
	w.c.filename = C.CString(cfg.Filename)
	w.c.count = C.int64_t(cfg.Count)

	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.HrlogWriter_Run, w.c),
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
