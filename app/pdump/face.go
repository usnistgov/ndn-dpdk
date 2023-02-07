package pdump

/*
#include "../../csrc/pdump/format.h"
#include "../../csrc/pdump/source.h"
#include "../../csrc/iface/face.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math"
	"sync"
	"unsafe"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/core/pcg32"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

// Direction indicates traffic direction.
type Direction string

// Direction values.
const (
	DirIncoming Direction = "RX"
	DirOutgoing Direction = "TX"
)

var dirImpls = map[Direction]struct {
	sllType C.rte_be16_t
	getRef  func(faceC *C.Face) *C.PdumpSourceRef
}{
	DirIncoming: {
		C.SLLIncoming,
		func(faceC *C.Face) *C.PdumpSourceRef { return &faceC.impl.rxPdump },
	},
	DirOutgoing: {
		C.SLLOutgoing,
		func(faceC *C.Face) *C.PdumpSourceRef { return &faceC.impl.txPdump },
	},
}

type faceDir struct {
	face iface.ID
	dir  Direction
}

func (fd faceDir) String() string {
	return fmt.Sprintf("%d-%s", fd.face, fd.dir)
}

func parseFaceDir(input string) (fd faceDir, e error) {
	_, e = fmt.Sscanf(input, "%d-%s", &fd.face, &fd.dir)
	return
}

var (
	faceSources     = map[faceDir]*FaceSource{}
	faceClosingOnce sync.Once
)

func handleFaceClosing(id iface.ID) {
	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()

	for dir := range dirImpls {
		s, ok := faceSources[faceDir{id, dir}]
		if !ok {
			continue
		}
		s.closeImpl()
	}
}

// FaceConfig contains FaceSource configuration.
type FaceConfig struct {
	Writer *Writer
	Face   iface.Face
	Dir    Direction
	Names  []NameFilterEntry
}

func (cfg *FaceConfig) validate() error {
	errs := []error{}

	if cfg.Writer == nil {
		errs = append(errs, errors.New("writer not found"))
	}

	if cfg.Face == nil {
		errs = append(errs, errors.New("face not found"))
	}

	if _, ok := dirImpls[cfg.Dir]; !ok {
		errs = append(errs, errors.New("invalid traffic direction"))
	}

	if n := len(cfg.Names); n == 0 || n > MaxNames {
		errs = append(errs, fmt.Errorf("must have between 1 and %d name filters", MaxNames))
	}

	for _, nf := range cfg.Names {
		if !(nf.SampleProbability >= 0.0 && nf.SampleProbability <= 1.0) {
			errs = append(errs, fmt.Errorf("sample probability of %s must be between 0.0 and 1.0", nf.Name))
		}
	}

	return errors.Join(errs...)
}

// NameFilterEntry matches a name prefix and specifies its sample rate.
// An empty name matches all packets.
type NameFilterEntry struct {
	Name              ndn.Name `json:"name" gqldesc:"NDN name prefix."`
	SampleProbability float64  `json:"sampleProbability" gqldesc:"Sample probability between 0.0 and 1.0." gqldflt:"1.0"`
}

// FaceSource is a packet dump source attached to a face on a single direction.
type FaceSource struct {
	FaceConfig
	key    faceDir
	logger *zap.Logger
	c      *C.PdumpFaceSource
}

func (s *FaceSource) setRef(expected, newPtr *C.PdumpSource) {
	ref := dirImpls[s.Dir].getRef((*C.Face)(s.Face.Ptr()))
	setSourceRef(ref, expected, newPtr)
}

// Close detaches the dump source.
func (s *FaceSource) Close() error {
	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()
	return s.closeImpl()
}

func (s *FaceSource) closeImpl() error {
	s.logger.Info("FaceSource close")
	s.setRef(&s.c.base, nil)
	delete(faceSources, s.key)

	go func() {
		urcu.Synchronize()
		s.Writer.stopSource()
		s.logger.Info("FaceSource freed")
		eal.Free(s.c)
	}()
	return nil
}

// NewFaceSource creates a FaceSource.
func NewFaceSource(cfg FaceConfig) (s *FaceSource, e error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	sourcesMutex.Lock()
	defer sourcesMutex.Unlock()

	s = &FaceSource{
		FaceConfig: cfg,
		key:        faceDir{cfg.Face.ID(), cfg.Dir},
	}
	if _, ok := faceSources[s.key]; ok {
		return nil, errors.New("another FaceSource is attached to this face and direction")
	}
	socket := s.Face.NumaSocket()

	s.logger = logger.With(s.Face.ID().ZapField("face"), zap.String("dir", string(s.Dir)))
	s.c = eal.Zmalloc[C.PdumpFaceSource]("PdumpFaceSource", C.sizeof_PdumpFaceSource, socket)
	s.c.base = C.PdumpSource{
		directMp: (*C.struct_rte_mempool)(pktmbuf.Direct.Get(socket).Ptr()),
		queue:    s.Writer.c.queue,
		filter:   C.PdumpSource_Filter(C.PdumpFaceSource_Filter),
		mbufType: MbufTypeSLL | C.uint32_t(dirImpls[s.Dir].sllType),
		mbufPort: C.uint16_t(s.Face.ID()),
		mbufCopy: true,
	}
	pcg32.Init(unsafe.Pointer(&s.c.rng))

	// sort by descending name length for longest prefix match
	slices.SortFunc(s.Names, func(a, b NameFilterEntry) bool { return len(a.Name) > len(b.Name) })
	prefixes := ndni.NewLNamePrefixFilterBuilder(unsafe.Pointer(&s.c.nameL), unsafe.Sizeof(s.c.nameL),
		unsafe.Pointer(&s.c.nameV), unsafe.Sizeof(s.c.nameV))
	for i, nf := range s.Names {
		if e := prefixes.Append(nf.Name); e != nil {
			eal.Free(s.c)
			return nil, errors.New("names too long")
		}
		s.c.sample[i] = C.uint32_t(math.Ceil(nf.SampleProbability * math.MaxUint32))
	}

	s.Writer.defineIntf(int(s.Face.ID()), pcapgo.NgInterface{
		Name:        fmt.Sprintf("face%d", s.Face.ID()),
		Description: iface.LocatorString(s.Face.Locator()),
		LinkType:    layers.LinkTypeLinuxSLL,
	})
	s.Writer.startSource()
	s.setRef(nil, &s.c.base)

	faceClosingOnce.Do(func() { iface.OnFaceClosing(handleFaceClosing) })
	faceSources[s.key] = s
	s.logger.Info("FaceSource open",
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(s.c))),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(s.Writer.c.queue))),
	)
	return s, nil
}
