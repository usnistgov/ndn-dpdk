package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var (
	lastTaskContextID int
	taskContextByID   = map[int]*TaskContext{}
	taskContextLock   sync.RWMutex
)

// TaskContext provides contextual information about an active fetch task.
type TaskContext struct {
	d        TaskDef
	id       int
	fetcher  *Fetcher
	w        *worker
	ts       *taskSlot
	stopping chan struct{}
}

// Counters returns congestion control and scheduling counters.
func (task *TaskContext) Counters() Counters {
	return task.ts.Logic().Counters()
}

// Stop aborts/stops the fetch task.
// This should be called even if the fetch task has succeeded.
func (task *TaskContext) Stop() {
	eal.CallMain(func() {
		task.w.RemoveTask(eal.MainReadSide, task.ts)
		task.ts.closeFd(task.d.FileSize)
		close(task.stopping)
		taskContextLock.Lock()
		defer taskContextLock.Unlock()
		delete(taskContextByID, task.id)
	})
}

// Finished determines if all segments have been fetched.
func (task *TaskContext) Finished() bool {
	return task.ts.Logic().Finished()
}

// TaskDef defines a fetch task that retrieves one segmented object.
type TaskDef struct {
	// InterestTemplateConfig contains the name prefix, InterestLifetime, etc.
	//
	// The fetcher neither retrieves metadata nor performs version discovery.
	// If the content is published with version component, it should appear in the name prefix.
	//
	// CanBePrefix and MustBeFresh are not normally used, but they may be included for benchmarking purpose.
	ndni.InterestTemplateConfig

	// SegmentRange specifies range of segment numbers.
	// If writing to a file, SegmentEnd must be explicitly specified.
	segmented.SegmentRange

	// Filename is the output file name.
	// If omitted, payload is not written to a file.
	Filename string `json:"filename,omitempty"`

	// FileSize is total payload length.
	// This is only relevant when writing to a file.
	// If set, the file will be truncated to this size after fetching is completed.
	FileSize *int64 `json:"fileSize,omitempty"`

	// SegmentLen is the payload length in each segment.
	// This is only needed when writing to a file.
	// If any segment has incorrect Content TLV-LENGTH, the output file would not contain correct payload.
	SegmentLen int `json:"segmentLen,omitempty"`
}

// TaskSlotConfig contains task slot configuration.
type TaskSlotConfig struct {
	// RxQueue configures the RX queue of Data packets going to each task slot.
	// CoDel cannot be used in these queues.
	RxQueue iface.PktQueueConfig `json:"rxQueue,omitempty"`

	// WindowCapacity is the maximum distance between lower and upper bounds of segment numbers in an ongoing fetch logic.
	WindowCapacity int `json:"windowCapacity,omitempty"`
}

func (cfg *TaskSlotConfig) applyDefaults() {
	cfg.RxQueue.DisableCoDel = true
	cfg.WindowCapacity = ringbuffer.AlignCapacity(cfg.WindowCapacity, 16, 65536)
}

type taskSlot C.FetchTask

// Init (re-)initializes the task slot to perform a fetch task.
// This should only be called on an inactive task slot.
func (ts *taskSlot) Init(d TaskDef) error {
	fl := ts.Logic()
	fl.Reset(d.SegmentRange)

	tpl := ndni.InterestTemplateFromPtr(unsafe.Pointer(&ts.tpl))
	d.InterestTemplateConfig.Apply(tpl)

	// FetchTask_DecodeData expects SegmentNameComponent TLV-TYPE at prefixV[prefixL]
	if uintptr(ts.tpl.prefixL+1) >= unsafe.Sizeof(ts.tpl.prefixV) {
		return errors.New("name too long")
	}
	ts.tpl.prefixV[ts.tpl.prefixL] = an.TtSegmentNameComponent

	logEntry := logger.With(
		zap.Int("slot-index", int(ts.index)),
		zap.Stringer("prefix", d.Prefix),
	)

	if d.Filename != "" {
		if d.SegmentLen <= 0 || d.SegmentLen > math.MaxUint32 {
			return errors.New("bad SegmentLen")
		}
		if d.SegmentEnd <= d.SegmentBegin || d.SegmentEnd > math.MaxUint32 {
			return errors.New("bad SegmentEnd")
		}

		fd, e := unix.Open(d.Filename, unix.O_WRONLY|unix.O_CREAT, 0o666)
		if e != nil {
			return fmt.Errorf("unix.Open(%s): %w", d.Filename, e)
		}

		logEntry = logEntry.With(
			zap.String("filename", d.Filename),
			zap.Int("fd", fd),
			zap.Int("segment-len", d.SegmentLen),
		)

		offsetBegin := int64(d.SegmentBegin) * int64(d.SegmentLen)
		offsetEnd := int64(d.SegmentEnd) * int64(d.SegmentLen)
		if e := unix.Fallocate(fd, 0, offsetBegin, offsetEnd-offsetBegin); e != nil {
			logEntry.Warn("unix.Fallocate error, this may affect write performance",
				zap.Int64("offset-begin", offsetBegin),
				zap.Int64("offset-end", offsetEnd),
				zap.Error(e),
			)
		}

		ts.fd, ts.segmentLen = C.int(fd), C.uint32_t(d.SegmentLen)
	}

	logEntry.Info("task init",
		zap.Uint64s("segment-range", []uint64{d.SegmentBegin, d.SegmentEnd}),
	)
	return nil
}

// RxQueueD returns the RX queue for Data packets.
func (ts *taskSlot) RxQueueD() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&ts.queueD))
}

// Logic returns the congestion control and scheduling logic.
func (ts *taskSlot) Logic() *Logic {
	return (*Logic)(&ts.logic)
}

func (ts *taskSlot) closeFd(fileSize *int64) {
	fd := int(ts.fd)
	if fd < 0 {
		return
	}

	logEntry := logger.With(
		zap.Int("slot-index", int(ts.index)),
		zap.Int("fd", fd),
		zap.Int64p("file-size", fileSize),
	)

	if fileSize != nil {
		if e := unix.Ftruncate(fd, *fileSize); e != nil {
			logEntry.Warn("unix.Ftruncate error",
				zap.Error(e),
			)
		}
	}

	if e := unix.Close(fd); e != nil {
		logEntry.Warn("unix.Close error",
			zap.Error(e),
		)
	}

	logEntry.Info("task output file closed")
	ts.fd = -1
}

func newTaskSlot(index int, cfg TaskSlotConfig, socket eal.NumaSocket) (ts *taskSlot) {
	ts = eal.Zmalloc[taskSlot]("FetchTask", unsafe.Sizeof(taskSlot{}), socket)
	*ts = taskSlot{
		fd:     -1,
		index:  C.uint8_t(index),
		worker: -1,
	}
	if e := ts.RxQueueD().Init(cfg.RxQueue, socket); e != nil {
		logger.Panic("TaskSlot.RxQueueD().Init error", zap.Error(e))
	}
	ts.Logic().Init(cfg.WindowCapacity, socket)
	return
}
