package ndn_test

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strconv"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/zyedidia/generic/mapset"
)

func TestLpFragmenter(t *testing.T) {
	assert, require := makeAR(t)

	data := ndn.MakeData("/D", bytes.Repeat([]byte{0xCC}, 3000))
	packet := data.ToPacket()
	packet.Lp.PitToken = bytesFromHex("808ECD3DF4E1B062")

	fragmenter := ndn.NewLpFragmenter(1000)
	frags, e := fragmenter.Fragment(packet)
	require.NoError(e)
	require.Len(frags, 4)

	for _, frag := range frags {
		wire, _ := tlv.EncodeFrom(frag)
		assert.LessOrEqual(len(wire), 1000)
	}

	tooSmall := ndn.NewLpFragmenter(10)
	_, e = tooSmall.Fragment(packet)
	assert.Error(e)
}

func TestLpReassembler(t *testing.T) {
	assert, require := makeAR(t)

	fragmenter := ndn.NewLpFragmenter(999)
	frames := [][]byte{}
	packetSet := mapset.New[int]()
	for i := 1000; i < 8000; i += 100 {
		packetSet.Put(i)
		data := ndn.MakeData(fmt.Sprint("/D/", i), bytes.Repeat([]byte{0xCC}, i))
		pkt := data.ToPacket()
		pkt.Lp.PitToken = []byte(strconv.Itoa(i))
		frags, _ := fragmenter.Fragment(pkt)
		for _, frag := range frags {
			wire, _ := tlv.EncodeFrom(frag)
			frames = append(frames, wire)
		}
	}
	rand.Shuffle(len(frames), reflect.Swapper(frames))

	reassembler := ndn.NewLpReassembler(80)
	for _, frame := range frames {
		var fragment ndn.Packet
		e := tlv.Decode(frame, &fragment)
		require.NoError(e)
		require.NotNil(fragment.Fragment)

		pkt, e := reassembler.Accept(&fragment)
		assert.NoError(e)
		if pkt != nil {
			require.NotNil(pkt.Data)
			i := len(pkt.Data.Content)
			pitTokenNum, _ := strconv.Atoi(string(pkt.Lp.PitToken))
			assert.Equal(i, pitTokenNum)
			assert.True(packetSet.Has(i))
			packetSet.Remove(i)
		}
	}
	assert.Equal(0, packetSet.Size())
}
