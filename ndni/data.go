package ndni

/*
#include "../csrc/ndn/data.h"
*/
import "C"
import (
	"crypto/sha256"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func (pdata *pData) getPtr() *C.PData {
	return (*C.PData)(unsafe.Pointer(pdata))
}

// Data represents a Data packet in mbuf.
type Data struct {
	m *Packet
	p *pData
}

// GetPacket converts Data to Packet.
func (data Data) GetPacket() *Packet {
	return data.m
}

func (data Data) String() string {
	return data.GetName().String()
}

// GetPDataPtr returns *C.PData pointer.
func (data Data) GetPDataPtr() unsafe.Pointer {
	return unsafe.Pointer(data.p)
}

// GetName returns Data name.
func (data Data) GetName() ndn.Name {
	return data.p.Name.ToName()
}

// GetFullName computes Data full name in Go.
func (data Data) GetFullName() ndn.Name {
	digest := data.ComputeDigest(false)
	name := data.GetName()
	name = append(name, ndn.MakeNameComponent(an.TtImplicitSha256DigestComponent, digest))
	return name
}

// GetFreshnessPeriod returns FreshnessPeriod.
func (data Data) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.p.FreshnessPeriod) * time.Millisecond
}

// CanSatisfy determines whether an Interest can satisfy this Data.
func (data Data) CanSatisfy(interest Interest) DataSatisfyResult {
	return DataSatisfyResult(C.PData_CanSatisfy(data.p.getPtr(), interest.p.getPtr()))
}

// GetDigest returns implicit digest computed in C.
func (data Data) GetDigest() []byte {
	if !data.p.HasDigest {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(&data.p.Digest[0]), sha256.Size)
}

// ComputeDigest computes implicit digest in Go.
func (data *Data) ComputeDigest(wantSave bool) []byte {
	d := sha256.Sum256(data.GetPacket().AsMbuf().ReadAll())
	if wantSave {
		data.p.HasDigest = true
		C.memcpy(unsafe.Pointer(&data.p.Digest[0]), unsafe.Pointer(&d[0]), sha256.Size)
	}
	return d[:]
}

// DigestPrepare prepares for computing implicit digest in C.
func (data *Data) DigestPrepare(op *cryptodev.Op) {
	C.DataDigest_Prepare(data.m.getPtr(), (*C.struct_rte_crypto_op)(op.GetPtr()))
}

// DataDigest_Finish finishes computing implicit digest in C.
func DataDigest_Finish(op *cryptodev.Op) (data *Data, e error) {
	if !op.IsSuccess() {
		return nil, op.Error()
	}

	npktC := C.DataDigest_Finish((*C.struct_rte_crypto_op)(op.GetPtr()))
	return PacketFromPtr(unsafe.Pointer(npktC)).AsData(), nil
}

type DataSatisfyResult int

const (
	DATA_SATISFY_YES         DataSatisfyResult = 0 // Data satisfies Interest
	DATA_SATISFY_NO          DataSatisfyResult = 1 // Data does not satisfy Interest
	DATA_SATISFY_NEED_DIGEST DataSatisfyResult = 2 // need Data digest to determine
)
