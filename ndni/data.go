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

// ToNData copies this packet into ndn.Data.
// Panics on error.
func (data Data) ToNData() ndn.Data {
	return *data.m.ToNPacket().Data
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

// SaveDigest computes and stores implicit digest in *C.PData.
func (data *Data) SaveDigest() {
	fullName := data.ToNData().FullName()
	digest := fullName[len(fullName)-1].Value
	copy(data.p.Digest[:], digest)
	data.p.HasDigest = true
}

// DigestPrepare prepares for computing implicit digest in C.
func (data *Data) DigestPrepare(op *cryptodev.Op) {
	C.DataDigest_Prepare(data.m.getPtr(), (*C.struct_rte_crypto_op)(op.GetPtr()))
}

// DataDigestFinish finishes computing implicit digest in C.
func DataDigestFinish(op *cryptodev.Op) (data *Data, e error) {
	if !op.IsSuccess() {
		return nil, op.Error()
	}

	npktC := C.DataDigest_Finish((*C.struct_rte_crypto_op)(op.GetPtr()))
	return PacketFromPtr(unsafe.Pointer(npktC)).AsData(), nil
}

// DataGen is a Data encoder optimized for traffic generator.
type DataGen C.DataGen

// NewDataGen creates a DataGen.
func NewDataGen(m *pktmbuf.Packet, suffix ndn.Name, freshnessPeriod time.Duration, content []byte) (gen *DataGen) {
	suffixV, _ := suffix.MarshalBinary()
	genC := C.DataGen_New((*C.struct_rte_mbuf)(m.GetPtr()),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), bytesToPtr(content))
	return (*DataGen)(genC)
}

// GetPtr returns *C.DataGen pointer.
func (gen *DataGen) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(gen)
}

func (gen *DataGen) getPtr() *C.DataGen {
	return (*C.DataGen)(gen)
}

// Close discards this DataGen.
func (gen *DataGen) Close() error {
	C.DataGen_Close(gen.getPtr())
	return nil
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(seg0, seg1 *pktmbuf.Packet, prefix ndn.Name) {
	prefixV, _ := prefix.MarshalBinary()
	C.DataGen_Encode_(gen.getPtr(),
		(*C.struct_rte_mbuf)(seg0.GetPtr()), (*C.struct_rte_mbuf)(seg1.GetPtr()),
		C.uint16_t(len(prefixV)), bytesToPtr(prefixV))
}
