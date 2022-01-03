package pdump

/*
#include "../../csrc/pdump/face.h"
#include "../../csrc/pdump/format.h"
#include "../../csrc/iface/face.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
	getRef  func(faceC *C.Face) *C.PdumpFaceRef
}{
	DirIncoming: {
		C.SLLIncoming,
		func(faceC *C.Face) *C.PdumpFaceRef { return &faceC.impl.rx.pdump },
	},
	DirOutgoing: {
		C.SLLOutgoing,
		func(faceC *C.Face) *C.PdumpFaceRef { return &faceC.impl.tx.pdump },
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
	faceSourcesLock sync.Mutex
	faceClosingOnce sync.Once
)

func handleFaceClosing(id iface.ID) {
	faceSourcesLock.Lock()
	defer faceSourcesLock.Unlock()

	for dir := range dirImpls {
		fs, ok := faceSources[faceDir{id, dir}]
		if !ok {
			continue
		}
		fs.closeImpl()
	}
}

// FaceConfig contains face dumper configuration.
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

	return multierr.Combine(errs...)
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
	c      *C.PdumpFace
}

func (fs *FaceSource) setPdumpFaceRef(expected, newPtr *C.PdumpFace) {
	ref := dirImpls[fs.Dir].getRef((*C.Face)(fs.Face.Ptr()))
	if old := C.PdumpFaceRef_Set(ref, newPtr); old != expected {
		fs.logger.Panic("PdumpFaceRef pointer mismatch",
			zap.Uintptr("new", uintptr(unsafe.Pointer(newPtr))),
			zap.Uintptr("old", uintptr(unsafe.Pointer(old))),
			zap.Uintptr("expected", uintptr(unsafe.Pointer(expected))),
		)
	}
}

// Close detaches the dump source.
func (fs *FaceSource) Close() error {
	faceSourcesLock.Lock()
	defer faceSourcesLock.Unlock()
	return fs.closeImpl()
}

func (fs *FaceSource) closeImpl() error {
	fs.logger.Info("PdumpFace close")
	fs.setPdumpFaceRef(fs.c, nil)
	delete(faceSources, fs.key)

	go func() {
		urcu.Synchronize()
		fs.Writer.stopSource()
		fs.logger.Info("PdumpFace freed")
		eal.Free(fs.c)
	}()
	return nil
}

// NewFaceSource creates a FaceSource.
func NewFaceSource(cfg FaceConfig) (fs *FaceSource, e error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}

	faceSourcesLock.Lock()
	defer faceSourcesLock.Unlock()

	fs = &FaceSource{
		FaceConfig: cfg,
		key:        faceDir{cfg.Face.ID(), cfg.Dir},
	}
	if _, ok := faceSources[fs.key]; ok {
		return nil, errors.New("another PdumpFace is attached to this face and direction")
	}
	socket := cfg.Face.NumaSocket()

	fs.logger = logger.With(cfg.Face.ID().ZapField("face"), zap.String("dir", string(cfg.Dir)))
	fs.c = (*C.PdumpFace)(eal.Zmalloc("PdumpFace", C.sizeof_PdumpFace, socket))
	*fs.c = C.PdumpFace{
		directMp: (*C.struct_rte_mempool)(pktmbuf.Direct.Get(socket).Ptr()),
		queue:    fs.Writer.c.queue,
		sllType:  dirImpls[cfg.Dir].sllType,
	}
	C.pcg32_srandom_r(&fs.c.rng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	// sort by decending name length for longest prefix match
	sort.Slice(cfg.Names, func(i, j int) bool { return len(cfg.Names[i].Name) > len(cfg.Names[j].Name) })
	prefixes := ndni.NewLNamePrefixFilterBuilder(unsafe.Pointer(&fs.c.nameL), unsafe.Sizeof(fs.c.nameL),
		unsafe.Pointer(&fs.c.nameV), unsafe.Sizeof(fs.c.nameV))
	for i, nf := range cfg.Names {
		if e := prefixes.Append(nf.Name); e != nil {
			eal.Free(fs.c)
			return nil, errors.New("names too long")
		}
		fs.c.sample[i] = C.uint32_t(math.Ceil(nf.SampleProbability * math.MaxUint32))
	}

	fs.Writer.defineFace(fs.Face)
	fs.Writer.startSource()
	fs.setPdumpFaceRef(nil, fs.c)

	faceClosingOnce.Do(func() { iface.OnFaceClosing(handleFaceClosing) })
	faceSources[fs.key] = fs

	fs.logger.Info("PdumpFace open",
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(fs.c))),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(fs.c.queue))),
	)
	return fs, nil
}
