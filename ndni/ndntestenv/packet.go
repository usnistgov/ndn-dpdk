package ndntestenv

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func makePacket(b []byte) *ndni.Packet {
	m := mbuftestenv.MakePacket(b)
	m.SetTimestamp(eal.TscNow())
	return ndni.PacketFromPtr(m.GetPtr())
}

func parseL2L3(pkt *ndni.Packet) {
	e := pkt.ParseL2()
	if e != nil {
		panic(e)
	}

	e = pkt.ParseL3(Name.Pool())
	if e != nil {
		panic(e)
	}
}

// MakeInterest creates an Interest packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: additional arguments to ndni.MakeInterest.
// Panics if packet constructed from bytes is not Interest.
func MakeInterest(input interface{}, args ...interface{}) *ndni.Interest {
	var pkt *ndni.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := Packet.Alloc()
			m.SetTimestamp(eal.TscNow())
			args = append(args, inp)
			interest, e := ndni.MakeInterest(m, args...)
			if e != nil {
				panic(e)
			}
			return interest
		}
		pkt = makePacket(testenv.BytesFromHex(inp))
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsInterest()
}

// MakeData creates a Data packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: additional arguments to ndni.MakeData.
// Panics if packet constructed from bytes is not Data.
func MakeData(input interface{}, args ...interface{}) *ndni.Data {
	var pkt *ndni.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			m := Packet.Alloc()
			m.SetTimestamp(eal.TscNow())
			data, e := ndni.MakeData(m, append([]interface{}{input}, args...)...)
			if e != nil {
				panic(e)
			}
			return data
		}
		pkt = makePacket(testenv.BytesFromHex(inp))
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsData()
}

// SetPort updates mbuf.port field.
func SetPort(pkt ndni.IL3Packet, port uint16) {
	pkt.GetPacket().AsMbuf().SetPort(port)
}

// GetPitToken returns the PIT token.
func GetPitToken(pkt ndni.IL3Packet) uint64 {
	return pkt.GetPacket().GetLpL3().PitToken
}

// SetPitToken updates the PIT token.
func SetPitToken(pkt ndni.IL3Packet, token uint64) {
	pkt.GetPacket().GetLpL3().PitToken = token
}

// CopyPitToken copies PIT token from src to pkt.
func CopyPitToken(pkt ndni.IL3Packet, src ndni.IL3Packet) {
	SetPitToken(pkt, GetPitToken(src))
}

// ClosePacket releases the mbuf.
func ClosePacket(pkt ndni.IL3Packet) {
	pkt.GetPacket().AsMbuf().Close()
}
