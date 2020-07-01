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
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func (pdata *pData) ptr() *C.PData {
	return (*C.PData)(unsafe.Pointer(pdata))
}

// Data represents a Data packet in mbuf.
type Data struct {
	m *Packet
	p *pData
}

// AsPacket converts Data to Packet.
func (data Data) AsPacket() *Packet {
	return data.m
}

// ToNData copies this packet into ndn.Data.
// Panics on error.
func (data Data) ToNData() ndn.Data {
	return *data.m.ToNPacket().Data
}

func (data Data) String() string {
	return data.Name().String()
}

// PDataPtr returns *C.PData pointer.
func (data Data) PDataPtr() unsafe.Pointer {
	return unsafe.Pointer(data.p)
}

// Name returns Data name.
func (data Data) Name() ndn.Name {
	return data.p.Name.ToName()
}

// FreshnessPeriod returns FreshnessPeriod.
func (data Data) FreshnessPeriod() time.Duration {
	return time.Duration(data.p.FreshnessPeriod) * time.Millisecond
}

// CanSatisfy determines whether an Interest can satisfy this Data.
func (data Data) CanSatisfy(interest Interest) DataSatisfyResult {
	return DataSatisfyResult(C.PData_CanSatisfy(data.p.ptr(), interest.p.ptr()))
}

// CachedImplicitDigest returns implicit digest computed in C.
func (data Data) CachedImplicitDigest() []byte {
	if !data.p.HasDigest {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(&data.p.Digest[0]), sha256.Size)
}

// ComputeImplicitDigest computes and stores implicit digest in *C.PData.
func (data *Data) ComputeImplicitDigest() []byte {
	fullName := data.ToNData().FullName()
	digest := fullName[len(fullName)-1].Value
	copy(data.p.Digest[:], digest)
	data.p.HasDigest = true
	return data.p.Digest[:]
}

// DigestPrepare prepares for computing implicit digest in C.
func (data *Data) DigestPrepare(op *cryptodev.Op) {
	C.DataDigest_Prepare(data.m.ptr(), (*C.struct_rte_crypto_op)(op.Ptr()))
}

// DataDigestFinish finishes computing implicit digest in C.
func DataDigestFinish(op *cryptodev.Op) (data *Data, e error) {
	if !op.IsSuccess() {
		return nil, op.Error()
	}

	npktC := C.DataDigest_Finish((*C.struct_rte_crypto_op)(op.Ptr()))
	return PacketFromPtr(unsafe.Pointer(npktC)).AsData(), nil
}

// DataGen is a Data encoder optimized for traffic generator.
type DataGen C.DataGen

// NewDataGen creates a DataGen.
func NewDataGen(m *pktmbuf.Packet, suffix ndn.Name, freshnessPeriod time.Duration, content []byte) (gen *DataGen) {
	suffixV, _ := suffix.MarshalBinary()
	genC := C.DataGen_New((*C.struct_rte_mbuf)(m.Ptr()),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), bytesToPtr(content))
	return (*DataGen)(genC)
}

// Ptr returns *C.DataGen pointer.
func (gen *DataGen) Ptr() unsafe.Pointer {
	return unsafe.Pointer(gen)
}

func (gen *DataGen) ptr() *C.DataGen {
	return (*C.DataGen)(gen)
}

// Close discards this DataGen.
func (gen *DataGen) Close() error {
	C.DataGen_Close(gen.ptr())
	return nil
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(seg0, seg1 *pktmbuf.Packet, prefix ndn.Name) {
	prefixV, _ := prefix.MarshalBinary()
	C.DataGen_Encode_(gen.ptr(),
		(*C.struct_rte_mbuf)(seg0.Ptr()), (*C.struct_rte_mbuf)(seg1.Ptr()),
		C.uint16_t(len(prefixV)), bytesToPtr(prefixV))
}
