package ndn

/*
#include "encode-data.h"

void
c_EncodeData1(struct rte_mbuf* m, uint16_t nameL, const uint8_t* nameV, struct rte_mbuf* payload)
{
	LName name = { .length = nameL, .value = nameV };
	EncodeData1(m, name, payload);
}

*/
import "C"
import "ndn-dpdk/dpdk"

func EncodeData1_GetHeadroom() int {
	return int(C.EncodeData1_GetHeadroom())
}

func EncodeData1_GetTailroom(nameLength int) int {
	return int(C.EncodeData1_GetTailroom(C.uint16_t(nameLength)))
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
	C.c_EncodeData1((*C.struct_rte_mbuf)(m1.GetPtr()), C.uint16_t(name.Size()), name.getValuePtr(),
		(*C.struct_rte_mbuf)(payload.GetPtr()))
	C.EncodeData2((*C.struct_rte_mbuf)(m2.GetPtr()), (*C.struct_rte_mbuf)(m1.GetPtr()))
	C.EncodeData3((*C.struct_rte_mbuf)(m1.GetPtr()))
	return dpdk.MbufFromPtr(m1.GetPtr()).AsPacket()
}
