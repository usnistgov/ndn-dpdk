package ndn

/*
#include "data.h"
*/
import "C"
import (
	"crypto/sha256"
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Data packet.
type Data struct {
	m Packet
	p *C.PData
}

func (data *Data) GetPacket() Packet {
	return data.m
}

func (data *Data) String() string {
	return data.GetName().String()
}

// Get *C.PData pointer.
func (data *Data) GetPDataPtr() unsafe.Pointer {
	return unsafe.Pointer(data.p)
}

func (data *Data) GetName() (n *Name) {
	n = new(Name)
	n.copyFromC(&data.p.name)
	return n
}

func (data *Data) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.p.freshnessPeriod) * time.Millisecond
}

func (data *Data) GetDigest() []byte {
	if !data.p.hasDigest {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(&data.p.digest[0]), sha256.Size)
}

func (data *Data) DigestPrepare(op dpdk.CryptoOp) {
	C.DataDigest_Prepare(data.m.c, (*C.struct_rte_crypto_op)(op.GetPtr()))
}

func DataDigest_Finish(op dpdk.CryptoOp) (data *Data, e error) {
	if status := op.GetStatus(); status != dpdk.CRYPTO_OP_SUCCESS {
		return nil, fmt.Errorf("crypto_op: %v", status)
	}

	npktC := C.DataDigest_Finish((*C.struct_rte_crypto_op)(op.GetPtr()))
	return PacketFromPtr(unsafe.Pointer(npktC)).AsData(), nil
}
