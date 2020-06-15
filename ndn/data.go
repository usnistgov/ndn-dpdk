package ndn

/*
#include "../csrc/ndn/data.h"
*/
import "C"
import (
	"crypto/sha256"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
)

// Data packet.
type Data struct {
	m *Packet
	p *C.PData
}

func (data *Data) GetPacket() *Packet {
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

// Compute Data full name (written in Go, for unit testing).
func (data *Data) GetFullName() (n *Name) {
	digest := data.ComputeDigest(false)
	nameV := data.GetName().GetValue()
	nameV = append(nameV, byte(TT_ImplicitSha256DigestComponent), byte(len(digest)))
	nameV = append(nameV, digest...)
	n, _ = NewName(nameV)
	return n
}

func (data *Data) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.p.freshnessPeriod) * time.Millisecond
}

type DataSatisfyResult int

const (
	DATA_SATISFY_YES         DataSatisfyResult = 0 // Data satisfies Interest
	DATA_SATISFY_NO          DataSatisfyResult = 1 // Data does not satisfy Interest
	DATA_SATISFY_NEED_DIGEST DataSatisfyResult = 2 // need Data digest to determine
)

func (data *Data) CanSatisfy(interest *Interest) DataSatisfyResult {
	return DataSatisfyResult(C.PData_CanSatisfy(data.p, interest.p))
}

func (data *Data) GetDigest() []byte {
	if !data.p.hasDigest {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(&data.p.digest[0]), sha256.Size)
}

// Compute Data digest (written in Go, for unit testing).
func (data *Data) ComputeDigest(wantSave bool) []byte {
	d := sha256.Sum256(data.GetPacket().AsMbuf().ReadAll())
	if wantSave {
		data.p.hasDigest = true
		C.memcpy(unsafe.Pointer(&data.p.digest[0]), unsafe.Pointer(&d[0]), sha256.Size)
	}
	return d[:]
}

func (data *Data) DigestPrepare(op *cryptodev.Op) {
	C.DataDigest_Prepare(data.m.getPtr(), (*C.struct_rte_crypto_op)(op.GetPtr()))
}

func DataDigest_Finish(op *cryptodev.Op) (data *Data, e error) {
	if !op.IsSuccess() {
		return nil, op.Error()
	}

	npktC := C.DataDigest_Finish((*C.struct_rte_crypto_op)(op.GetPtr()))
	return PacketFromPtr(unsafe.Pointer(npktC)).AsData(), nil
}
