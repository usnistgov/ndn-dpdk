package hrlog

/*
#include "../../csrc/hrlog/writer.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"path"
	"sync"
	"time"
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
	ErrDisabled = errors.New("hrlog module disabled")
	ErrWriter   = errors.New("writer failed")
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
		id:     fmt.Sprintf("%s %d", cfg.Filename, time.Now().UnixNano()),
		cfg:    cfg,
		finish: make(chan error, 1),
	}
	task.stop = ealthread.InitStopFlag(unsafe.Pointer(&task.stopC))

	w.tasks.Store(task.id, task)
	go func() {
		w.queue <- task
	}()
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
	for task := range w.queue {
		logger.Info("writer open", zap.String("filename", task.cfg.Filename))
		task.execute(capacity)
		w.tasks.Delete(task.id)
		logger.Info("writer close", zap.String("filename", task.cfg.Filename))
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

// Task is a pending or ongoing hrlog collection task.
type Task struct {
	id     string
	cfg    TaskConfig
	stopC  C.ThreadStopFlag
	stop   ealthread.StopFlag
	finish chan error
}

// Stop stops a collection task.
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
