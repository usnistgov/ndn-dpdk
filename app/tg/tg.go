// Package tg controls traffic generator elements.
package tg

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Thread roles.
const (
	RoleInput    = iface.RoleRx
	RoleOutput   = iface.RoleTx
	RoleConsumer = tgdef.RoleConsumer
	RoleProducer = tgdef.RoleProducer
)

var logger = logging.New("tg")

var (
	mapFaceGen      = map[iface.ID]*TrafficGen{}
	mapFaceGenMutex sync.RWMutex
)

func saveChooseRxlTxl() (undo func()) {
	oldChooseRxLoop, oldChooseTxLoop := iface.ChooseRxLoop, iface.ChooseTxLoop
	return func() { iface.ChooseRxLoop, iface.ChooseTxLoop = oldChooseRxLoop, oldChooseTxLoop }
}

// TrafficGen represents the traffic generator on a face.
type TrafficGen struct {
	face    iface.Face
	rxl     []iface.RxLoop
	txl     iface.TxLoop
	workers []ealthread.ThreadWithRole

	producer   *tgproducer.Producer
	fileServer *fileserver.Server
	consumer   *tgconsumer.Consumer
	fetcher    *fetch.Fetcher
	exit       chan struct{}
}

// Face returns the face on which this traffic generator operates.
func (gen TrafficGen) Face() iface.Face {
	return gen.face
}

// Producer returns the producer module.
func (gen TrafficGen) Producer() *tgproducer.Producer {
	return gen.producer
}

// FileServer returns the file server module.
func (gen TrafficGen) FileServer() *fileserver.Server {
	return gen.fileServer
}

// Consumer returns the fixed rate consumer module.
func (gen TrafficGen) Consumer() *tgconsumer.Consumer {
	return gen.consumer
}

// Fetcher returns the congestion aware fetcher module.
func (gen TrafficGen) Fetcher() *fetch.Fetcher {
	return gen.fetcher
}

// Workers implements tggql.withCommonFields interface.
func (gen TrafficGen) Workers() []ealthread.ThreadWithRole {
	return gen.workers
}

// Launch launches the traffic generator.
func (gen *TrafficGen) Launch() error {
	ealthread.Launch(gen.txl)
	for _, rxl := range gen.rxl {
		ealthread.Launch(rxl)
	}

	if gen.producer != nil {
		gen.producer.Launch()
	} else if gen.fileServer != nil {
		gen.fileServer.Launch()
	}

	if gen.consumer != nil {
		gen.consumer.Launch()
	} else if gen.fetcher != nil {
		gen.fetcher.Launch()
	}

	return nil
}

func (gen *TrafficGen) configureDemux() {
	gen.face.EnableInputDemuxes()
	demuxI, demuxD, demuxN := gen.face.DemuxOf(ndni.PktInterest), gen.face.DemuxOf(ndni.PktData), gen.face.DemuxOf(ndni.PktNack)

	if gen.producer != nil {
		gen.producer.ConnectRxQueues(demuxI)
	} else if gen.fileServer != nil {
		gen.fileServer.ConnectRxQueues(demuxI)
	}

	if gen.consumer != nil {
		gen.consumer.ConnectRxQueues(demuxD, demuxN)
	} else if gen.fetcher != nil {
		gen.fetcher.ConnectRxQueues(demuxD, demuxN)
	}
}

// Stop stops the traffic generator.
// It can be launched again.
func (gen *TrafficGen) Stop() error {
	errs := []error{}
	if gen.producer != nil {
		errs = append(errs, gen.producer.Stop())
	}
	if gen.consumer != nil {
		errs = append(errs, gen.consumer.Stop())
	}
	if gen.fetcher != nil {
		errs = append(errs, gen.fetcher.Stop())
	}
	return errors.Join(errs...)
}

// Close releases resources.
// It cannot be launched again.
func (gen *TrafficGen) Close() error {
	if e := gen.Stop(); e != nil {
		return e
	}
	close(gen.exit)

	if gen.face != nil {
		mapFaceGenMutex.Lock()
		delete(mapFaceGen, gen.face.ID())
		mapFaceGenMutex.Unlock()
	}

	errs := []error{}
	gatherCloseErr := func(c io.Closer) {
		if c != nil && !reflect.ValueOf(c).IsNil() {
			errs = append(errs, c.Close())
		}
	}
	gatherCloseErr(gen.producer)
	gatherCloseErr(gen.fileServer)
	gatherCloseErr(gen.consumer)
	gatherCloseErr(gen.fetcher)
	gatherCloseErr(gen.face)
	for _, rxl := range gen.rxl {
		gatherCloseErr(rxl)
	}
	gatherCloseErr(gen.txl)

	var lcores eal.LCores
	for _, w := range gen.workers {
		if lc := w.LCore(); lc.Valid() {
			lcores = append(lcores, lc)
		}
	}
	ealthread.AllocFree(lcores...)

	*gen = TrafficGen{}
	return errors.Join(errs...)
}

// New creates a traffic generator.
func New(cfg Config) (gen *TrafficGen, e error) {
	if e = cfg.Validate(); e != nil {
		return nil, e
	}

	gen = &TrafficGen{
		exit: make(chan struct{}),
	}
	defer func(gen *TrafficGen) {
		if e != nil {
			gen.Close()
		}
	}(gen)

	// mutex also prevents parallel creations causing ChooseRx/TxLoop conflict
	mapFaceGenMutex.Lock()
	defer mapFaceGenMutex.Unlock()

	iface.RxParseFor = ndni.ParseForApp
	defer saveChooseRxlTxl()()
	iface.ChooseRxLoop = func(rxg iface.RxGroup) iface.RxLoop {
		if _, ok := rxg.(iface.RxGroupSingleFace); !ok {
			return nil
		}
		rxl := iface.NewRxLoop(rxg.NumaSocket())
		gen.rxl = append(gen.rxl, rxl)
		gen.workers = append(gen.workers, rxl)
		return rxl
	}
	iface.ChooseTxLoop = func(face iface.Face) iface.TxLoop {
		gen.txl = iface.NewTxLoop(face.NumaSocket())
		gen.workers = append(gen.workers, gen.txl)
		return gen.txl
	}

	if gen.face, e = cfg.Face.CreateFace(); e != nil {
		return nil, fmt.Errorf("error creating face %w", e)
	}
	if len(gen.rxl) == 0 {
		logger.Warn("face creation did not result in dedicated RxLoop creation; this face is incompatible with traffic generator: results are inaccurate, closing any traffic generator may cause a crash")
	}
	if gen.txl == nil {
		logger.Warn("face creation did not result in dedicated TxLoop creation; this face is incompatible with traffic generator: results are inaccurate, closing any traffic generator may cause a crash")
	}

	if cfg.Producer != nil {
		producer, e := tgproducer.New(gen.face, *cfg.Producer)
		if e != nil {
			return nil, fmt.Errorf("error creating producer %w", e)
		}
		gen.workers = append(gen.workers, producer.Workers()...)
		gen.producer = producer
	}
	if cfg.FileServer != nil {
		fileServer, e := fileserver.New(gen.face, *cfg.FileServer)
		if e != nil {
			return nil, fmt.Errorf("error creating fileServer %w", e)
		}
		gen.workers = append(gen.workers, fileServer.Workers()...)
		gen.fileServer = fileServer
	}

	if cfg.Consumer != nil {
		consumer, e := tgconsumer.New(gen.face, *cfg.Consumer)
		if e != nil {
			return nil, fmt.Errorf("error creating consumer %w", e)
		}
		gen.workers = append(gen.workers, consumer.Workers()...)
		gen.consumer = consumer
	}
	if cfg.Fetcher != nil {
		fetcher, e := fetch.New(gen.face, *cfg.Fetcher)
		if e != nil {
			return nil, fmt.Errorf("error creating fetcher %w", e)
		}
		gen.workers = append(gen.workers, fetcher.Workers()...)
		gen.fetcher = fetcher
	}

	gen.configureDemux()
	if e := ealthread.AllocThread(gen.workers...); e != nil {
		return nil, fmt.Errorf("error allocating gen.workers %w", e)
	}

	mapFaceGen[gen.face.ID()] = gen
	return gen, nil
}

// Get retrieves traffic generator instance by face.
func Get(id iface.ID) *TrafficGen {
	mapFaceGenMutex.RLock()
	defer mapFaceGenMutex.RUnlock()
	return mapFaceGen[id]
}
