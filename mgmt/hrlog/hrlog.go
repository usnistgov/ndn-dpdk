package hrlog

/*
#include "writer.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/ringbuffer"
)

const ringCapacity = 65536

// Initialize high resolution logger.
func Init() error {
	r, e := ringbuffer.New("theHrlogRing", ringCapacity, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return e
	}
	C.theHrlogRing = (*C.struct_rte_ring)(r.GetPtr())
	go collectLoop()
	return nil
}

// Management module for high resolution logger.
type HrlogMgmt struct{}

func (HrlogMgmt) Start(args StartArgs, reply *struct{}) error {
	collectJobsLock.Lock()
	defer collectJobsLock.Unlock()
	if _, ok := collectJobs[args.Filename]; ok {
		return errors.New("duplicate collect job")
	}

	job := new(collectJob)
	job.StartArgs = args
	if job.Count == 0 {
		job.Count = 1 << 28 // 268 million samples, 2GB file
	}
	eal.InitStopFlag(unsafe.Pointer(&job.stop))
	job.finish = make(chan error, 1)

	collectJobs[args.Filename] = job
	collectStart <- job
	return nil
}

func (HrlogMgmt) Stop(args FilenameArg, reply *struct{}) error {
	job := func() *collectJob {
		collectJobsLock.Lock()
		defer collectJobsLock.Unlock()
		return collectJobs[args.Filename]
	}()
	if job == nil {
		return errors.New("job not found")
	}

	stop := eal.InitStopFlag(unsafe.Pointer(&job.stop))
	stop.BeforeWait()
	e := <-job.finish
	stop.AfterWait()

	collectJobsLock.Lock()
	defer collectJobsLock.Unlock()
	delete(collectJobs, args.Filename)
	return e
}

type FilenameArg struct {
	Filename string
}

type StartArgs struct {
	FilenameArg
	Count int
}

type collectJob struct {
	StartArgs
	stop   C.ThreadStopFlag
	finish chan error
}

var (
	collectJobsLock sync.Mutex
	collectJobs     = make(map[string]*collectJob)
	collectStart    = make(chan *collectJob)
)

func collectLoop() {
	for {
		job := <-collectStart
		filenameC := C.CString(job.Filename)
		defer C.free(unsafe.Pointer(filenameC))

		res := C.Hrlog_RunWriter(filenameC, ringCapacity, C.int(job.Count), &job.stop)
		if res != 0 {
			job.finish <- fmt.Errorf("Hrlog_RunWriter error %d", res)
		} else {
			job.finish <- nil
		}
	}
}
