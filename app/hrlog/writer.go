package hrlog

/*
#include "../../csrc/hrlog/writer.h"
*/
import "C"
import (
	"context"
	"errors"
	"path"
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
	RingCapacity int `json:"ringCapacity,omitempty"`
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

// Writer is a hrlog writer thread.
type Writer struct {
	ealthread.Thread
	ring  *ringbuffer.Ring
	queue chan *writerTask
}

var _ ealthread.ThreadWithRole = (*Writer)(nil)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Writer) ThreadRole() string {
	return Role
}

// Submit submits a task.
func (w *Writer) Submit(ctx context.Context, cfg TaskConfig) (res chan error) {
	cfg.applyDefaults()
	task := &writerTask{
		ctx:    ctx,
		cfg:    cfg,
		finish: make(chan bool, 1),
	}

	res = make(chan error, 1)
	select {
	case w.queue <- task:
	default:
		res <- ErrQueueFull
		return
	}

	go func() {
		select {
		case <-ctx.Done():
			res <- ctx.Err()
		case ok := <-task.finish:
			if ok {
				res <- nil
			} else {
				res <- ErrWriter
			}
		}
	}()
	return
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
		if task.ctx.Err() != nil {
			continue
		}
		logger.Info("writer open", zap.String("filename", task.cfg.Filename))
		task.execute(capacity)
		logger.Info("writer close", zap.String("filename", task.cfg.Filename))
	}
	logger.Info("writer shutdown")
}

// NewWriter creates a hrlog writer thread.
func NewWriter(cfg WriterConfig) (w *Writer, e error) {
	w = &Writer{
		queue: make(chan *writerTask, 256),
	}

	w.ring, e = ringbuffer.New(ringbuffer.AlignCapacity(cfg.RingCapacity, 64, 65536),
		eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return nil, e
	}

	w.Thread = ealthread.New(cptr.Func0.Void(w.loop), ealthread.NewStopClose(w.queue))
	return w, e
}

type writerTask struct {
	ctx    context.Context
	cfg    TaskConfig
	finish chan bool
}

func (task *writerTask) execute(nSkip int) {
	c := (*C.HrlogWriter)(eal.Zmalloc("HrlogWriter", C.sizeof_HrlogWriter, eal.NumaSocket{}))
	*c = C.HrlogWriter{
		filename: C.CString(task.cfg.Filename),
		nSkip:    C.int(nSkip),
		nTotal:   C.int(task.cfg.Count),
	}
	ctrl := ealthread.InitCtrl(unsafe.Pointer(&c.ctrl))
	defer func() {
		ctrl = nil
		C.free(unsafe.Pointer(c.filename))
		eal.Free(c)
	}()

	finish := make(chan bool, 1)
	go func() { finish <- bool(C.Hrlog_RunWriter(c)) }()

	select {
	case <-task.ctx.Done():
		break
	case ok := <-finish:
		task.finish <- ok
		return
	}

	stop := ctrl.Stopper()
	stop.BeforeWait()
	<-finish
	stop.AfterWait()
}
