package ifacetest

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func TestInOrderReassembler(t *testing.T) {
	assert, require := makeAR(t)

	reassembler := iface.NewInOrderReassembler()

	steps := []struct {
		input  string
		output string
	}{
		{"6414 seq=5108A0A1A2A3A4A5A600 fragindex=520100 fragcount=530102 payload=5002B0B1",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 payload=5002B2B3",
			"B0B1B2B3"}, // accepted, delivering
		{"6414 seq=5108A0A1A2A3A4A5A611 fragindex=520101 fragcount=530102 payload=5002C2C3",
			""}, // out of order (not first fragment)
		{"6414 seq=5108A0A1A2A3A4A5A620 fragindex=520100 fragcount=530103 payload=5002D0D1",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A622 fragindex=520102 fragcount=530103 payload=5002D6D7",
			""}, // out of order
		{"6414 seq=5108A0A1A2A3A4A5A621 fragindex=520101 fragcount=530103 payload=5002D2D3",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A622 fragindex=520102 fragcount=530103 payload=5002D4D5",
			"D0D1D2D3D4D5"}, // accepted, delivering
		{"6414 seq=5108A0A1A2A3A4A5A630 fragindex=520100 fragcount=530102 payload=5002E0E1",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A640 fragindex=520100 fragcount=530102 payload=5002F0F1",
			""}, // accepted, discarding buffer
		{"6414 seq=5108A0A1A2A3A4A5A641 fragindex=520101 fragcount=530102 payload=5002F2F3",
			"F0F1F2F3"}, // accepted, delivering
	}
	for _, step := range steps {
		fragPkt := ndn.PacketFromDpdk(dpdktestenv.PacketFromHex(step.input))
		e := fragPkt.ParseL2()
		require.NoError(e, step.input)

		reassPkt := reassembler.Receive(fragPkt)
		if step.output == "" {
			assert.True(reassPkt.GetPtr() == nil, step.input)
		} else if assert.NotNil(reassPkt.GetPtr(), step.input) {
			payload := reassPkt.AsDpdkPacket().ReadAll()
			assert.Equal(dpdktestenv.BytesFromHex(step.output), payload, step.input)
		}
	}

	counters := reassembler.ReadCounters()
	assert.Equal(uint64(8), counters.Accepted)
	assert.Equal(uint64(2), counters.OutOfOrder)
	assert.Equal(uint64(3), counters.Delivered)
	assert.Equal(uint64(1), counters.Incomplete)
}
