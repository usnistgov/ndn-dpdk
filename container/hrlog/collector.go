package hrlog

/*
#include "../../csrc/hrlog/writer.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

// Error conditions.
var (
	ErrDisabled  = errors.New("hrlog module disabled")
	ErrDuplicate = errors.New("duplicate filename")
)

// Config contains job configuration.
type Config struct {
	Filename string `json:"filename"`
	Count    int    `json:"count,omitempty"`
}

func (cfg *Config) applyDefaults() {
	if cfg.Count <= 0 {
		cfg.Count = 1 << 28 // 268 million samples, 2GB file
	}
}

// Collector is a hrlog collection task.
type Collector struct {
	cfg    Config
	stopC  C.ThreadStopFlag
	stop   ealthread.StopFlag
	finish chan error
}

// Start starts a collection job.
func Start(cfg Config) (c *Collector, e error) {
	if C.theHrlogRing == nil {
		return nil, ErrDisabled
	}

	cfg.applyDefaults()
	c = &Collector{
		cfg:    cfg,
		finish: make(chan error, 1),
	}
	c.stop = ealthread.InitStopFlag(unsafe.Pointer(&c.stopC))

	collectorLock.Lock()
	defer collectorLock.Unlock()
	if collectorMap[c.cfg.Filename] != nil {
		return nil, ErrDuplicate
	}

	collectorMap[c.cfg.Filename] = c
	collectQueue <- c
	return c, nil
}

// Stop stops a collection job.
func (c *Collector) Stop() (e error) {
	c.stop.BeforeWait()
	e = <-c.finish
	c.stop.AfterWait()

	collectorLock.Lock()
	defer collectorLock.Unlock()
	delete(collectorMap, c.cfg.Filename)

	return e
}

func (c *Collector) execute() {
	filenameC := C.CString(c.cfg.Filename)
	defer C.free(unsafe.Pointer(filenameC))

	res := C.Hrlog_RunWriter(filenameC, ringCapacity, C.int(c.cfg.Count), &c.stopC)
	if res != 0 {
		c.finish <- fmt.Errorf("Hrlog_RunWriter error %d", res)
	} else {
		c.finish <- nil
	}
}

// Init initializes the high resolution logger.
func Init() {
	initOnce.Do(func() {
		r, e := ringbuffer.New(ringCapacity, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
		if e != nil {
			panic(e)
		}
		C.theHrlogRing = (*C.struct_rte_ring)(r.Ptr())
		go collectLoop()
	})
}

const ringCapacity = 65536

var (
	initOnce      sync.Once
	collectorMap  = make(map[string]*Collector)
	collectorLock sync.Mutex
	collectQueue  = make(chan *Collector)
)

func collectLoop() {
	for c := range collectQueue {
		c.execute()
	}
}
