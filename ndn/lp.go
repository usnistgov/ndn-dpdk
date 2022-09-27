package ndn

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"strconv"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/zyedidia/generic/cache"
	"golang.org/x/exp/slices"
)

func lpIsCritical(typ uint32) bool {
	return typ < 800 || typ > 959 || typ&0x03 != 0
}

const fragmentOverhead = 0 +
	1 + 3 + // LpPacket TL
	1 + 1 + 8 + // LpSeqNum
	1 + 1 + 2 + // FragIndex
	1 + 1 + 2 + // FragCount
	1 + 3 + // LpPayload TL
	0

// LpL3 contains layer 3 fields in NDNLPv2 header.
type LpL3 struct {
	PitToken   []byte
	NackReason uint8
	CongMark   uint8
}

// Empty returns true if LpL3 has zero fields.
func (lph LpL3) Empty() bool {
	return len(lph.PitToken) == 0 && lph.NackReason == an.NackNone && lph.CongMark == 0
}

func (lph LpL3) encode() (fields []tlv.Field) {
	if len(lph.PitToken) > 0 {
		fields = append(fields, tlv.TLVBytes(an.TtPitToken, lph.PitToken))
	}
	switch lph.NackReason {
	case an.NackNone:
	case an.NackUnspecified:
		fields = append(fields, tlv.TLV(an.TtNack))
	default:
		fields = append(fields, tlv.TLV(an.TtNack, tlv.TLVNNI(an.TtNackReason, lph.NackReason)))
	}
	if lph.CongMark != 0 {
		fields = append(fields, tlv.TLVNNI(an.TtCongestionMark, lph.CongMark))
	}
	return fields
}

func (lph *LpL3) inheritFrom(src LpL3) {
	lph.PitToken = slices.Clone(src.PitToken)
	lph.CongMark = src.CongMark
}

// LpFragment represents an NDNLPv2 fragmented frame.
type LpFragment struct {
	SeqNum    uint64
	FragIndex int
	FragCount int
	Header    []byte // encoded NDNLPv2 L3 header fields
	Payload   []byte // LpPayload TLV-VALUE
}

var _ tlv.Fielder = LpFragment{}

func (frag LpFragment) String() string {
	return strconv.FormatUint(frag.SeqNum, 16) + ":" + strconv.Itoa(frag.FragIndex) + ":" + strconv.Itoa(frag.FragCount)
}

// Field implements tlv.Fielder interface.
func (frag LpFragment) Field() tlv.Field {
	if frag.FragIndex < 0 || frag.FragIndex >= frag.FragCount {
		return tlv.FieldError(ErrFragment)
	}

	var seqNum [8]byte
	binary.BigEndian.PutUint64(seqNum[:], frag.SeqNum)
	return tlv.TLV(an.TtLpPacket,
		tlv.TLVBytes(an.TtLpSeqNum, seqNum[:]),
		tlv.TLVNNI(an.TtFragIndex, frag.FragIndex),
		tlv.TLVNNI(an.TtFragCount, frag.FragCount),
		tlv.Bytes(frag.Header),
		tlv.TLVBytes(an.TtLpPayload, frag.Payload))
}

// LpFragmenter splits Packet into fragments.
type LpFragmenter struct {
	nextSeqNum uint64
	room       int
}

// Fragment fragments a packet.
func (fragmenter *LpFragmenter) Fragment(full *Packet) (frags []*Packet, e error) {
	header, payload, e := full.encodeL3()
	if e != nil {
		return nil, e
	}
	sizeofFirstFragment := fragmenter.room - len(header)
	if sizeofPayload := len(payload); sizeofFirstFragment > sizeofPayload { // no fragmentation necessary
		return []*Packet{full}, nil
	}

	if sizeofFirstFragment <= 0 { // MTU is too small to fit this packet
		return nil, ErrFragment
	}

	var first Packet
	first.Lp = full.Lp
	first.Fragment = &LpFragment{
		Header:  header,
		Payload: payload[:sizeofFirstFragment],
	}
	frags = append(frags, &first)

	for offset, nextOffset := sizeofFirstFragment, 0; offset < len(payload); offset = nextOffset {
		nextOffset = offset + fragmenter.room
		if nextOffset > len(payload) {
			nextOffset = len(payload)
		}

		var frag Packet
		frag.Fragment = &LpFragment{
			Payload: payload[offset:nextOffset],
		}
		frags = append(frags, &frag)
	}

	for i, frag := range frags {
		frag.Fragment.SeqNum = fragmenter.nextSeqNum
		fragmenter.nextSeqNum++
		frag.Fragment.FragIndex = i
		frag.Fragment.FragCount = len(frags)
	}
	return frags, nil
}

// NewLpFragmenter creates a LpFragmenter.
func NewLpFragmenter(mtu int) *LpFragmenter {
	var fragmenter LpFragmenter
	fragmenter.nextSeqNum = rand.Uint64()
	fragmenter.room = mtu - fragmentOverhead
	return &fragmenter
}

// LpReassembler reassembles fragments.
type LpReassembler struct {
	pms *cache.Cache[uint64, *lpPartialPacket]
}

// Accept processes a fragment.
// pkt.Fragment must not be nil.
func (reass *LpReassembler) Accept(pkt *Packet) (full *Packet, e error) {
	seq0 := pkt.Fragment.SeqNum - uint64(pkt.Fragment.FragIndex)

	pp, _ := reass.pms.Get(seq0)
	if pp == nil {
		pp = &lpPartialPacket{}
		reass.pms.Put(seq0, pp)
	}

	full, discard, e := pp.Accept(pkt)
	if discard {
		reass.pms.Remove(seq0)
	}

	return full, e
}

// NewLpReassembler creates a LpReassembler.
func NewLpReassembler(capacity int) *LpReassembler {
	return &LpReassembler{
		pms: cache.New[uint64, *lpPartialPacket](capacity),
	}
}

type lpPartialPacket struct {
	lpl3     LpL3
	buffer   [][]byte
	accepted int
}

func (pp *lpPartialPacket) Accept(pkt *Packet) (full *Packet, discard bool, e error) {
	if pp.accepted == 0 {
		pp.buffer = make([][]byte, pkt.Fragment.FragCount)
		pp.acceptOne(pkt)
		return nil, false, nil
	}

	if pkt.Fragment.FragCount != len(pp.buffer) {
		return nil, true, ErrFragment
	}
	if pp.buffer[pkt.Fragment.FragIndex] != nil {
		return nil, false, ErrFragment
	}

	pp.acceptOne(pkt)
	if pp.accepted == len(pp.buffer) {
		full, e = pp.reassemble()
		return full, true, e
	}
	return nil, false, nil
}

func (pp *lpPartialPacket) acceptOne(pkt *Packet) {
	if pkt.Fragment.FragIndex == 0 {
		pp.lpl3 = pkt.Lp
	}
	pp.buffer[pkt.Fragment.FragIndex] = pkt.Fragment.Payload
	pp.accepted++
}

func (pp *lpPartialPacket) reassemble() (full *Packet, e error) {
	full = &Packet{Lp: pp.lpl3}

	payload := bytes.Join(pp.buffer, nil)
	if e = full.decodePayload(payload); e != nil {
		return nil, e
	}

	return full, nil
}
