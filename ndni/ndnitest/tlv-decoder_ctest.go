package ndnitest

/*
#include "../../csrc/ndni/tlv-decoder.h"
*/
import "C"
import (
	"bytes"
	"math"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func ctestTlvDecoderReadSkip(t *testing.T) {
	assert, _ := makeAR(t)

	scratch := (*C.uint8_t)(C.malloc(8))
	defer C.free(unsafe.Pointer(scratch))

	var d C.TlvDecoder
	C.TlvDecoder_Read(&d, scratch, 0)
	C.TlvDecoder_Skip(&d, 0)
	assert.EqualValues(0, d.length)

	p := makePacket("B0B1B2B3", "B4B5B6B7", "C0C1C2C3C4C5C6C7")
	defer p.Close()
	C.TlvDecoder_Init(&d, p.mbuf)
	assert.EqualValues(16, d.length)
	assert.EqualValues(0, d.offset)

	r := C.TlvDecoder_Read(&d, scratch, 3)
	assert.EqualValues(13, d.length)
	assert.EqualValues(3, d.offset)
	assert.NotSame(scratch, r)
	assert.Equal(bytesFromHex("B0B1B2"), C.GoBytes(unsafe.Pointer(r), 3))

	r = C.TlvDecoder_Read(&d, scratch, 2)
	assert.EqualValues(11, d.length)
	assert.EqualValues(1, d.offset)
	assert.Same(scratch, r)
	assert.Equal(bytesFromHex("B3B4"), C.GoBytes(unsafe.Pointer(r), 2))

	C.TlvDecoder_Skip(&d, 1)
	assert.EqualValues(10, d.length)
	assert.EqualValues(2, d.offset)

	C.TlvDecoder_Skip(&d, 2)
	assert.EqualValues(8, d.length)
	assert.EqualValues(0, d.offset)
}

func ctestTlvDecoderClone(t *testing.T) {
	assert, require := makeAR(t)

	p := makePacket("", "A0A1A2A3", "B0B1B2", "", "C0C1", "D0")
	defer p.Close()
	const pktlen = 10
	require.Equal(pktlen, p.Len())
	payload := bytesFromHex("A0A1A2A3B0B1B2C0C1D0")

	for offset := 0; offset <= pktlen; offset++ {
		for count := 1; count < pktlen-offset; count++ {
			var d C.TlvDecoder
			C.TlvDecoder_Init(&d, p.mbuf)
			C.TlvDecoder_Skip(&d, C.uint32_t(offset))

			clone := C.TlvDecoder_Clone(&d, C.uint32_t(count), (*C.struct_rte_mempool)(mbuftestenv.Indirect.Pool().Ptr()), nil)
			if !assert.NotNil(clone, "%d-%d", offset, count) {
				continue
			}
			for seg := clone; seg != nil; seg = seg.next {
				assert.NotZero(seg.data_len, "%d-%d", offset, count)
			}
			clonePkt := pktmbuf.PacketFromPtr(unsafe.Pointer(clone))
			assert.Equal(count, clonePkt.Len(), "%d-%d", offset, count)
			assert.Equal(payload[offset:offset+count], clonePkt.Bytes(), "%d-%d", offset, count)
			clonePkt.Close()
		}
	}
}

func ctestTlvDecoderLinearize(t *testing.T) {
	assert, _ := makeAR(t)

	var d C.TlvDecoder
	C.TlvDecoder_Linearize(&d, 0)
	assert.EqualValues(0, d.length)

	p := makePacket("B0B1B2B3", "B4B5B6B7", "C0C1C2C3C4C5C6C7")
	defer p.Close()
	C.TlvDecoder_Init(&d, p.mbuf)
	C.TlvDecoder_Skip(&d, 1)

	// contiguous
	r := C.TlvDecoder_Linearize(&d, 2)
	assert.Equal(bytesFromHex("B1B2"), C.GoBytes(unsafe.Pointer(r), 2))
	assert.EqualValues(3, d.p.nb_segs)
	assert.EqualValues(13, d.length)
	assert.EqualValues(3, d.offset)

	// move to first
	r = C.TlvDecoder_Linearize(&d, 6)
	assert.Equal(bytesFromHex("B3B4B5B6B7C0"), C.GoBytes(unsafe.Pointer(r), 6))
	C.rte_mbuf_sanity_check(p.mbuf, 1)
	assert.NotZero(d.p.data_off)
	assert.EqualValues(2, d.p.nb_segs)
	assert.EqualValues(7, d.length)
	assert.EqualValues(0, d.offset)

	p = makePacket(mbuftestenv.Headroom(directDataroom-5), "B0B1B2B3", "C0C1C2C3")
	defer p.Close()
	C.TlvDecoder_Init(&d, p.mbuf)
	C.TlvDecoder_Skip(&d, 1)

	// move to first with memmove
	r = C.TlvDecoder_Linearize(&d, 7)
	assert.Equal(bytesFromHex("B1B2B3C0C1C2C3"), C.GoBytes(unsafe.Pointer(r), 7))
	C.rte_mbuf_sanity_check(p.mbuf, 1)
	assert.EqualValues(1, d.p.nb_segs)
	assert.Zero(d.p.data_off)
	assert.Nil(d.m)
	assert.EqualValues(0, d.length)
	assert.EqualValues(0, d.offset)

	p = makePacket(mbuftestenv.Headroom(0), bytes.Repeat([]byte{0xA0}, directDataroom), bytes.Repeat([]byte{0xA1}, directDataroom))
	defer p.Close()
	C.TlvDecoder_Init(&d, p.mbuf)
	C.TlvDecoder_Skip(&d, 2)

	// copy to new
	r = C.TlvDecoder_Linearize(&d, C.uint16_t(directDataroom-1))
	assert.Equal(append(bytes.Repeat([]byte{0xA0}, directDataroom-2), 0xA1), C.GoBytes(unsafe.Pointer(r), C.int(directDataroom-1)))
	C.rte_mbuf_sanity_check(p.mbuf, 1)
	assert.EqualValues(3, d.p.nb_segs)
	assert.EqualValues(2, d.p.data_len)
	assert.EqualValues(directDataroom-1, d.p.next.data_len)
	assert.EqualValues(directDataroom-1, d.p.next.next.data_len)
	assert.EqualValues(directDataroom-1, d.length)
	assert.EqualValues(0, d.offset)
}

func ctestTlvDecoderTL(t *testing.T) {
	assert, _ := makeAR(t)

	for _, tt := range ndntestvector.TlvElementTests {
		p := makePacket(tt.Input)
		defer p.Close()
		var d C.TlvDecoder
		C.TlvDecoder_Init(&d, p.mbuf)

		var length C.uint32_t
		typ := C.TlvDecoder_ReadTL(&d, &length)

		if tt.Bad {
			assert.Zero(typ, tt.Input)
		} else {
			assert.EqualValues(tt.Type, typ, tt.Input)
			value := bytesFromHex(tt.Value)
			assert.EqualValues(len(value), length, tt.Input)
			assert.EqualValues(len(value), d.length, tt.Input)

			var nni C.uint64_t
			ok := C.TlvDecoder_ReadNni(&d, length, math.MaxUint64, &nni)
			if bool(ok) {
				assert.EqualValues(tt.Nni, nni, tt.Input)
			} else {
				assert.True(tt.Nni == ndntestvector.NotNni, tt.Input)
			}
		}

	}
}

func ctestTlvDecoderValueDecoder(t *testing.T) {
	assert, _ := makeAR(t)

	var d, vd C.TlvDecoder
	C.TlvDecoder_MakeValueDecoder(&d, 0, &vd)
	assert.EqualValues(0, vd.length)

	p := makePacket("07 04 C0C1", "C2C3")
	defer p.Close()
	C.TlvDecoder_Init(&d, p.mbuf)

	var length C.uint32_t
	typ := C.TlvDecoder_ReadTL(&d, &length)
	assert.EqualValues(0x07, typ)
	assert.EqualValues(4, length)
	assert.EqualValues(2, d.offset)
	assert.EqualValues(4, d.length)

	C.TlvDecoder_MakeValueDecoder(&d, length, &vd)
	assert.EqualValues(2, vd.offset)
	assert.EqualValues(4, vd.length)
	assert.EqualValues(0, d.length)
}
