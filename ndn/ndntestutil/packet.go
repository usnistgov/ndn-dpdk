package ndntestutil

import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func makePacket(b []byte) (pkt ndn.Packet) {
	m := dpdktestenv.PacketFromBytes(b)
	m.SetTimestamp(dpdk.TscNow())
	pkt = ndn.PacketFromPtr(m.GetPtr())
	return pkt
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

var interestTpl = ndn.NewInterestTemplate()

// Make Interest on dpdktestenv DirectMp.
// input: packet bytes as []byte or HEX, or name URI.
// args: additional arguments to ndn.MakeInterest.
// Panics if packet constructed from bytes is not Interest.
func MakeInterest(input interface{}, args ...interface{}) *ndn.Interest {
	var pkt ndn.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
			m.AsPacket().SetTimestamp(dpdk.TscNow())
			interest, e := ndn.MakeInterest(m, inp, args...)
			if e != nil {
				panic(e)
			}
			return interest
		} else {
			pkt = makePacket(dpdktestenv.BytesFromHex(inp))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsInterest()
}

// Make Data on dpdktestenv DirectMp.
// input: packet bytes as []byte or HEX, or name URI.
// args: additional arguments to ndn.MakeData.
// Panics if packet constructed from bytes is not Data.
func MakeData(input interface{}, args ...interface{}) *ndn.Data {
	var pkt ndn.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
			m.AsPacket().SetTimestamp(dpdk.TscNow())
			data, e := ndn.MakeData(m, inp, args...)
			if e != nil {
				panic(e)
			}
			return data
		} else {
			pkt = makePacket(dpdktestenv.BytesFromHex(inp))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsData()
}

func SetFaceId(pkt ndn.IL3Packet, port uint16) {
	pkt.GetPacket().AsDpdkPacket().SetPort(port)
}

func GetPitToken(pkt ndn.IL3Packet) uint64 {
	return pkt.GetPacket().GetLpL3().GetPitToken()
}

func SetPitToken(pkt ndn.IL3Packet, token uint64) {
	pkt.GetPacket().GetLpL3().SetPitToken(token)
}

func CopyPitToken(pkt ndn.IL3Packet, src ndn.IL3Packet) {
	SetPitToken(pkt, GetPitToken(src))
}

func ClosePacket(pkt ndn.IL3Packet) {
	pkt.GetPacket().AsDpdkPacket().Close()
}
