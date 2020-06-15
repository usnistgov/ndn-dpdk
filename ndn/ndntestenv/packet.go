package ndntestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func makePacket(b []byte) *ndn.Packet {
	m := mbuftestenv.MakePacket(b)
	m.SetTimestamp(eal.TscNow())
	return ndn.PacketFromPtr(m.GetPtr())
}

func parseL2L3(pkt *ndn.Packet) {
	e := pkt.ParseL2()
	if e != nil {
		panic(e)
	}

	e = pkt.ParseL3(Name.Pool())
	if e != nil {
		panic(e)
	}
}

// Make Interest on dpdktestenv DirectMp.
// input: packet bytes as []byte or HEX, or name URI.
// args: additional arguments to ndn.MakeInterest.
// Panics if packet constructed from bytes is not Interest.
func MakeInterest(input interface{}, args ...interface{}) *ndn.Interest {
	var pkt *ndn.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := Packet.Alloc()
			m.SetTimestamp(eal.TscNow())
			args = append(args, inp)
			interest, e := ndn.MakeInterest(m, args...)
			if e != nil {
				panic(e)
			}
			return interest
		} else {
			pkt = makePacket(mbuftestenv.BytesFromHex(inp))
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
	var pkt *ndn.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := Packet.Alloc()
			m.SetTimestamp(eal.TscNow())
			data, e := ndn.MakeData(m, inp, args...)
			if e != nil {
				panic(e)
			}
			return data
		} else {
			pkt = makePacket(mbuftestenv.BytesFromHex(inp))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsData()
}

func SetFaceId(pkt ndn.IL3Packet, port uint16) {
	pkt.GetPacket().AsMbuf().SetPort(port)
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
	pkt.GetPacket().AsMbuf().Close()
}
