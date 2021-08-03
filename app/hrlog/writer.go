package hrlog

/*
#include "../../csrc/hrlog/writer.h"
*/
import "C"
import (
	"errors"
	"path"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"go.uber.org/zap"
)

// Role is writer thread role name.
const Role = "HRLOG"

// TheWriter is the current Writer instance.
var TheWriter *Writer

var logger = logging.New("hrlog")

// Error conditions.
var (
	ErrDisabled  = errors.New("hrlog module disabled")
	ErrDuplicate = errors.New("duplicate filename")
	ErrWriter    = errors.New("writer failed")
)

// WriterConfig contains writer configuration.
type WriterConfig struct {
	RingCapacity int
}

// Writer is a hrlog writer thread.
type Writer struct {
	ealthread.Thread
	ring  *ringbuffer.Ring
	tasks sync.Map // map[filename]*Task
	queue chan *Task
	stop  ealthread.StopClose
}

var _ ealthread.ThreadWithRole = (*Writer)(nil)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Writer) ThreadRole() string {
	return Role
}

// Submit submits a task.
func (w *Writer) Submit(cfg TaskConfig) (task *Task, e error) {
	cfg.applyDefaults()
	task = &Task{
		cfg:    cfg,
		finish: make(chan error, 1),
	}
	task.stop = ealthread.InitStopFlag(unsafe.Pointer(&task.stopC))

	_, loaded := w.tasks.LoadOrStore(task.cfg.Filename, task)
	if loaded {
		return nil, ErrDuplicate
	}
	w.queue <- task
	return task, nil
}

func (w *Writer) loop() {
	TheWriter, C.theHrlogRing = w, (*C.struct_rte_ring)(w.ring.Ptr())
	defer func() {
		TheWriter, C.theHrlogRing = nil, nil
		w.ring.Close()
	}()

	capacity := w.ring.Capacity()
	logger.Info("writer ready", w.LCore().ZapField("lc"), zap.Int("capacity", capacity))
	for c := range w.queue {
		logger.Info("writer open", zap.String("filename", c.cfg.Filename))
		c.execute(capacity)
		w.tasks.Delete(c.cfg.Filename)
		logger.Info("writer close", zap.String("filename", c.cfg.Filename))
	}
	logger.Info("writer shutdown")
}

// NewWriter creates a hrlog writer thread.
func NewWriter(cfg WriterConfig) (w *Writer, e error) {
	w = &Writer{
		queue: make(chan *Task),
	}

	w.ring, e = ringbuffer.New(ringbuffer.AlignCapacity(cfg.RingCapacity, 64, 65536),
		eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return nil, e
	}

	w.Thread = ealthread.New(cptr.Func0.Void(w.loop), ealthread.NewStopClose(w.queue))
	return w, e
}

// TaskConfig contains task configuration.
type TaskConfig struct {
	Filename string `json:"filename"`
	Count    int    `json:"count,omitempty"`
}

func (cfg *TaskConfig) applyDefaults() {
	cfg.Filename = path.Clean(cfg.Filename)
	if cfg.Count <= 0 {
		cfg.Count = 1 << 28 // 268 million samples, 2GB file
	}
}

// Task is an ongoing hrlog collection task.
type Task struct {
	cfg    TaskConfig
	stopC  C.ThreadStopFlag
	stop   ealthread.StopFlag
	finish chan error
}

// Stop stops a collection job.
func (task *Task) Stop() (e error) {
	task.stop.BeforeWait()
	e = <-task.finish
	task.stop.AfterWait()
	return e
}

func (task *Task) execute(nSkip int) {
	filenameC := C.CString(task.cfg.Filename)
	defer C.free(unsafe.Pointer(filenameC))

	ok := bool(C.Hrlog_RunWriter(filenameC, C.int(nSkip), C.int(task.cfg.Count), &task.stopC))
	if ok {
		task.finish <- nil
	} else {
		task.finish <- ErrWriter
	}
}

// Post posts entries to the hrlog collector.
func Post(entries []uint64) {
	ptr, count := cptr.ParseCptrArray(entries)
	C.Hrlog_Post((*C.HrlogEntry)(ptr), C.uint16_t(count))
}
