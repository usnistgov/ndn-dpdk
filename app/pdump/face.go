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

	"github.com/jwangsadinata/go-multimap"
	"github.com/jwangsadinata/go-multimap/setmultimap"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var (
	faceDumps       multimap.MultiMap = setmultimap.New()
	faceClosingOnce sync.Once
)

func handleFaceClosing(id iface.ID) {
	pds, _ := faceDumps.Get(id)
	for _, pd := range pds {
		pd.(*Face).Close()
	}
}

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

// FaceConfig contains face dumper configuration.
type FaceConfig struct {
	ID    string            `json:"id"` // GraphQL face ID
	Dir   Direction         `json:"dir"`
	Names []NameFilterEntry `json:"names"`
}

func (cfg FaceConfig) validate() error {
	errs := []error{}
	if _, ok := dirImpls[cfg.Dir]; !ok {
		errs = append(errs, errors.New("invalid traffic direction"))
	}
	if n := len(cfg.Names); n == 0 || n > MaxNames {
		errs = append(errs, fmt.Errorf("must have between 1 and %d name filters", MaxNames))
	}
	for i, nf := range cfg.Names {
		if !(nf.SampleRate >= 0.0 && nf.SampleRate <= 1.0) {
			errs = append(errs, fmt.Errorf("sample rate at index %d must be between 0.0 and 1.0", i))
		}
	}
	return multierr.Combine(errs...)
}

// NameFilterEntry matches a name prefix and specifies its sample rate.
// An empty name matches all packets.
type NameFilterEntry struct {
	Name       ndn.Name `json:"name"`
	SampleRate float64  `json:"sampleRate"`
}

// Face is a packet dumper attached to a face on a single direction.
type Face struct {
	face iface.Face
	dir  Direction
	w    *Writer
	c    *C.PdumpFace
	cr   *C.PdumpFaceRef
}

// Close detaches the dumper.
func (pd *Face) Close() error {
	logger.Info("PdumpFace close",
		pd.face.ID().ZapField("face"),
		zap.String("dir", string(pd.dir)),
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(pd.c))),
	)

	if ptr := C.PdumpFaceRef_Set(pd.cr, nil); ptr != pd.c {
		panic(fmt.Errorf("PdumpFaceRef pointer mismatch %p %p", ptr, pd.c))
	}
	faceDumps.Remove(pd.face.ID(), pd)

	go func() {
		urcu.Synchronize()
		pd.w.stopDumper()
		logger.Info("PdumpFace freed", zap.Uintptr("dumper", uintptr(unsafe.Pointer(pd.c))))
		eal.Free(pd.c)
	}()
	return nil
}

// DumpFace attaches a packet dumper on a face.
func DumpFace(face iface.Face, w *Writer, cfg FaceConfig) (pd *Face, e error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}
	// a zero-length name (i.e. capture all packets) should appear first
	sort.Slice(cfg.Names, func(i, j int) bool { return len(cfg.Names[i].Name) < len(cfg.Names[j].Name) })

	socket := face.NumaSocket()
	dirImpl := dirImpls[cfg.Dir]

	pd = &Face{
		face: face,
		dir:  cfg.Dir,
		w:    w,
		c:    (*C.PdumpFace)(eal.Zmalloc("PdumpFace", C.sizeof_PdumpFace, socket)),
		cr:   dirImpl.getRef((*C.Face)(face.Ptr())),
	}
	pd.c.directMp = (*C.struct_rte_mempool)(pktmbuf.Direct.Get(socket).Ptr())
	pd.c.queue = w.c.queue
	C.pcg32_srandom_r(&pd.c.rng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	pd.c.sllType = dirImpl.sllType

	nameBuf := cptr.AsByteSlice(&pd.c.nameV)
	for i, nf := range cfg.Names {
		nameV, _ := nf.Name.MarshalBinary()
		pd.c.sample[i] = C.uint32_t(math.Ceil(nf.SampleRate * math.MaxUint32))
		pd.c.nameL[i] = C.uint16_t(len(nameV))
		copy(nameBuf, nameV)
		nameBuf = nameBuf[len(nameV):]
	}

	w.AddFace(face)
	w.startDumper()

	if ptr := C.PdumpFaceRef_Set(pd.cr, pd.c); ptr != nil {
		panic(fmt.Errorf("PdumpFaceRef pointer mismatch %p != nil", ptr))
	}

	faceClosingOnce.Do(func() { iface.OnFaceClosing(handleFaceClosing) })
	faceDumps.Put(face.ID(), pd)

	logger.Info("PdumpFace open",
		face.ID().ZapField("face"),
		zap.String("dir", string(pd.dir)),
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(pd.c))),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(pd.c.queue))),
	)
	return pd, nil
}
