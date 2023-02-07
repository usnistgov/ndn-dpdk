package pdump

/*
#include "../../csrc/pdump/source.h"
#include "../../csrc/ethface/rxtable.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"go.uber.org/zap"
)

// EthGrab indicates an opportunity to grab packets from Ethernet port.
type EthGrab string

// EthGrab values.
const (
	EthGrabRxUnmatched EthGrab = "RxUnmatched"
)

var ethPortSources = map[*ethport.Port]*EthPortSource{}

// EthPortConfig contains EthPortSource configuration.
type EthPortConfig struct {
	Writer *Writer
	Port   *ethport.Port
	Grab   EthGrab

	rxt *C.EthRxTable
}

func (cfg *EthPortConfig) validate() error {
	errs := []error{}

	if cfg.Writer == nil {
		errs = append(errs, errors.New("writer not found"))
	}

	if cfg.Port == nil {
		errs = append(errs, errors.New("port not found"))
	} else if cfg.rxt = (*C.EthRxTable)(ethport.RxTablePtrFromPort(cfg.Port)); cfg.rxt == nil {
		errs = append(errs, errors.New("port is not using RxTable"))
	}

	if cfg.Grab != EthGrabRxUnmatched {
		errs = append(errs, errors.New("grab not supported"))
	}

	return errors.Join(errs...)
}

// EthPortSource is a packet dump source attached to an Ethernet port on a grab opportunity.
type EthPortSource struct {
	EthPortConfig
	logger *zap.Logger
	c      *C.PdumpSource
}

func (s *EthPortSource) setRef(expected, newPtr *C.PdumpSource) {
	setSourceRef(&s.rxt.pdumpUnmatched, expected, newPtr)
}

// Close detaches the dump source.
func (s *EthPortSource) Close() error {
	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()
	return s.closeImpl()
}

func (s *EthPortSource) closeImpl() error {
	s.logger.Info("EthPortSource close")
	s.setRef(s.c, nil)
	delete(ethPortSources, s.Port)

	go func() {
		urcu.Synchronize()
		s.Writer.stopSource()
		s.logger.Info("EthPortSource freed")
		eal.Free(s.c)
	}()
	return nil
}

// NewEthPortSource creates a EthPortSource.
func NewEthPortSource(cfg EthPortConfig) (s *EthPortSource, e error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()

	s = &EthPortSource{
		EthPortConfig: cfg,
	}
	if _, ok := ethPortSources[s.Port]; ok {
		return nil, errors.New("another EthPortSource is attached to this port")
	}
	dev := s.Port.EthDev()
	id, socket := dev.ID(), dev.NumaSocket()

	s.logger = logger.With(dev.ZapField("port"))
	s.c = eal.Zmalloc[C.PdumpSource]("PdumpSource", C.sizeof_PdumpSource, socket)
	*s.c = C.PdumpSource{
		directMp: (*C.struct_rte_mempool)(pktmbuf.Direct.Get(socket).Ptr()),
		queue:    s.Writer.c.queue,
		filter:   nil,
		mbufType: MbufTypeRaw,
		mbufPort: C.uint16_t(id),
		mbufCopy: false,
	}

	s.Writer.defineIntf(id, pcapgo.NgInterface{
		Name:        fmt.Sprintf("port%d", id),
		Description: dev.Name(),
		LinkType:    layers.LinkTypeEthernet,
	})
	s.Writer.startSource()
	s.setRef(nil, s.c)

	ethPortSources[s.Port] = s
	s.logger.Info("EthPortSource open",
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(s.c))),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(s.Writer.c.queue))),
	)
	return s, nil
}

func init() {
	if ethdev.MaxEthDevs > iface.MinID {
		panic("FaceID and EthDevID must not overlap")
	}
}
