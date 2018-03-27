package ndntestutil

import (
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

// Parse NDN name from URI.
// Panics if URI is malformed.
func ParseName(uri string) *ndn.Name {
	name, e := ndn.ParseName(uri)
	if e != nil {
		panic(e)
	}
	return name
}

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
// Panics if packet constructed from bytes is not Interest.
func MakeInterest(input interface{}) *ndn.Interest {
	var pkt ndn.Packet
	switch input1 := input.(type) {
	case []byte:
		pkt = makePacket(input1)
	case string:
		if input1[0] == '/' {
			m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
			interestTpl.Encode(m, ParseName(input1), nil)
			pkt = ndn.PacketFromDpdk(m)
		} else {
			pkt = makePacket(dpdktestenv.BytesFromHex(input1))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsInterest()
}

// Make Data on dpdktestenv DirectMp.
// input: packet bytes as []byte or HEX, or name URI.
// Panics if packet constructed from bytes is not Data.
func MakeData(input interface{}) *ndn.Data {
	var pkt ndn.Packet
	switch input1 := input.(type) {
	case []byte:
		pkt = makePacket(input1)
	case string:
		if input1[0] == '/' {
			m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
			ndn.EncodeData(m, ParseName(input1), 0, nil)
			pkt = ndn.PacketFromDpdk(m)
		} else {
			pkt = makePacket(dpdktestenv.BytesFromHex(input1))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsData()
}

// Make Nack on dpdktestenv DirectMp.
// input: packet bytes as []byte or HEX, or "name~reason".
// Panics if packet constructed from bytes is not Nack.
func MakeNack(input interface{}) *ndn.Nack {
	var pkt ndn.Packet
	switch input1 := input.(type) {
	case []byte:
		pkt = makePacket(input1)
	case string:
		if input1[0] == '/' {
			nackReason := ndn.NackReason_Unspecified
			if reasonPos := strings.LastIndexByte(input1, '~'); reasonPos >= 0 {
				nackReason = ndn.ParseNackReason(input1[reasonPos+1:])
				input1 = input1[:reasonPos]
			}
			m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
			interestTpl.Encode(m, ParseName(input1), nil)
			pkt = ndn.PacketFromDpdk(m)
			parseL2L3(pkt)
			return ndn.MakeNackFromInterest(pkt.AsInterest(), nackReason)
		} else {
			pkt = makePacket(dpdktestenv.BytesFromHex(input1))
		}
	default:
		panic("unrecognized input type")
	}

	parseL2L3(pkt)
	return pkt.AsNack()
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
