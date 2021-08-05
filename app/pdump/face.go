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
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
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

var dirSLL = map[Direction]C.rte_be16_t{
	DirIncoming: C.SLLIncoming,
	DirOutgoing: C.SLLOutgoing,
}

// FaceConfig contains face dumper configuration.
type FaceConfig struct {
	Dir   Direction         `json:"dir"`
	Names []NameFilterEntry `json:"names"`
}

func (cfg FaceConfig) validate() error {
	errs := []error{}
	if _, ok := dirSLL[cfg.Dir]; !ok {
		errs = append(errs, errors.New("invalid traffic direction"))
	}
	if n := len(cfg.Names); n == 0 || n > MaxNames {
		errs = append(errs, fmt.Errorf("must have between 1 and %d name filters", MaxNames))
	}
	for i, nf := range cfg.Names {
		if !(nf.SampleRate >= 0.0 && nf.SampleRate <= 1.0) {
			errs = append(errs, fmt.Errorf("sample rate %d must be between 0.0 and 1.0", i))
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

// Process submits packets for potential dumping.
func (pd *Face) Process(pkts pktmbuf.Vector) {
	C.PdumpFace_Process(pd.c, C.FaceID(pd.face.ID()), (**C.struct_rte_mbuf)(pkts.Ptr()), C.uint16_t(len(pkts)))
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
	socket := face.NumaSocket()

	pd = &Face{
		face: face,
		dir:  cfg.Dir,
		w:    w,
		c:    (*C.PdumpFace)(eal.Zmalloc("PdumpFace", C.sizeof_PdumpFace, socket)),
	}
	pd.c.directMp = (*C.struct_rte_mempool)(pktmbuf.Direct.Get(socket).Ptr())
	pd.c.queue = w.c.queue
	C.pcg32_srandom_r(&pd.c.rng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	pd.c.sllType = dirSLL[pd.dir]

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

	faceC := (*C.Face)(face.Ptr())
	switch pd.dir {
	case DirIncoming:
		pd.cr = &faceC.impl.rx.pdump
	case DirOutgoing:
		pd.cr = &faceC.impl.tx.pdump
	}
	if ptr := C.PdumpFaceRef_Set(pd.cr, pd.c); ptr != nil {
		panic(fmt.Errorf("PdumpFaceRef pointer mismatch %p != nil", ptr))
	}

	logger.Info("PdumpFace open",
		face.ID().ZapField("face"),
		zap.String("dir", string(pd.dir)),
		zap.Uintptr("dumper", uintptr(unsafe.Pointer(pd.c))),
		zap.Uintptr("queue", uintptr(unsafe.Pointer(pd.c.queue))),
	)
	return pd, nil
}
