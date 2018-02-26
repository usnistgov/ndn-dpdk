package ndntestutil

import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func ParseName(nameStr string) *ndn.Name {
	name, e := ndn.ParseName(nameStr)
	if e != nil {
		panic(e)
	}
	return name
}

func MakePacket(input interface{}) ndn.Packet {
	var b []byte
	switch input1 := input.(type) {
	case []byte:
		b = input1
	case string:
		b = dpdktestenv.BytesFromHex(input1)
	}
	pkt := dpdktestenv.PacketFromBytes(b)
	pkt.SetTimestamp(dpdk.TscNow())
	return ndn.PacketFromPtr(pkt.GetPtr())
}

func parseL2L3(pkt ndn.Packet) {
	e := pkt.ParseL2()
	if e != nil {
		panic(e)
	}

	e = pkt.ParseL3(dpdktestenv.GetMp(dpdktestenv.MPID_DIRECT))
	if e != nil {
		panic(e)
	}
}

func MakeInterest(input interface{}) *ndn.Interest {
	pkt := MakePacket(input)
	parseL2L3(pkt)
	return pkt.AsInterest()
}

func MakeData(input interface{}) *ndn.Data {
	pkt := MakePacket(input)
	parseL2L3(pkt)
	return pkt.AsData()
}

func MakeNack(input interface{}) *ndn.Nack {
	pkt := MakePacket(input)
	parseL2L3(pkt)
	return pkt.AsNack()
}

type iNdnPacket interface {
	GetPacket() ndn.Packet
}

func SetFaceId(pkt iNdnPacket, port uint16) {
	pkt.GetPacket().AsDpdkPacket().SetPort(port)
}

func ClosePacket(pkt iNdnPacket) {
	pkt.GetPacket().AsDpdkPacket().Close()
}
