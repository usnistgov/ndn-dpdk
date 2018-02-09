package ndn

/*
#include "data-pkt.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

type DataPkt struct {
	c C.DataPkt
}

// Test whether the decoder may contain a Data.
func (d *TlvDecodePos) IsData() bool {
	return d.it.PeekOctet() == int(TT_Data)
}

// Decode a Data.
func (d *TlvDecodePos) ReadData() (data DataPkt, e error) {
	res := C.DecodeData(d.getPtr(), &data.c)
	if res != C.NdnError_OK {
		return DataPkt{}, NdnError(res)
	}
	return data, nil
}

func (data *DataPkt) GetName() *Name {
	return (*Name)(unsafe.Pointer(&data.c.name))
}

func (data *DataPkt) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.c.freshnessPeriod) * time.Millisecond
}

func EncodeData1_GetHeadroom() int {
	return int(C.EncodeData1_GetHeadroom())
}

func EncodeData1_GetTailroom(name *Name) int {
	return int(C.EncodeData1_GetTailroom(&name.c))
}

func EncodeData1_GetTailroomMax() int {
	return int(C.EncodeData1_GetTailroomMax())
}

func EncodeData2_GetHeadroom() int {
	return int(C.EncodeData2_GetHeadroom())
}

func EncodeData2_GetTailroom() int {
	return int(C.EncodeData2_GetTailroom())
}

// Make a Data.
func EncodeData(name *Name, payload dpdk.IMbuf, m1 dpdk.IMbuf, m2 dpdk.IMbuf) dpdk.Packet {
	C.EncodeData1((*C.struct_rte_mbuf)(m1.GetPtr()), &name.c, (*C.struct_rte_mbuf)(payload.GetPtr()))
	C.EncodeData2((*C.struct_rte_mbuf)(m2.GetPtr()), (*C.struct_rte_mbuf)(m1.GetPtr()))
	C.EncodeData3((*C.struct_rte_mbuf)(m1.GetPtr()))
	return dpdk.MbufFromPtr(m1.GetPtr()).AsPacket()
}
