package ifacetest

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func TestInOrderReassembler(t *testing.T) {
	assert, require := makeAR(t)

	reassembler := iface.InOrderReassembler{}

	steps := []struct {
		input  string
		output string
	}{
		{"6414 seq=5108A0A1A2A3A4A5A600 fragindex=520100 fragcount=530102 payload=5002B0B1",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 payload=5002B2B3",
			"B0B1B2B3"}, // accepted, delivering
		{"6414 seq=5108A0A1A2A3A4A5A611 fragindex=520101 fragcount=530102 payload=5002C2C3",
			""}, // not first fragment
		{"6414 seq=5108A0A1A2A3A4A5A620 fragindex=520100 fragcount=530103 payload=5002D0D1",
			""}, // accepted
		{"6414 seq=5108A0A1A2A3A4A5A622 fragindex=520102 fragcount=530103 payload=5002D4D5 first",
			""}, // missing fragindex=1, discarding buffer
		{"6414 seq=5108A0A1A2A3A4A5A621 fragindex=520101 fragcount=530103 payload=5002D2D3",
			""}, // dropping because buffer discarded
		{"6414 seq=5108A0A1A2A3A4A5A622 fragindex=520102 fragcount=530103 payload=5002D4D5 second",
			""}, // dropping because buffer discarded
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

	counters := reassembler.GetCounters()
	assert.EqualValues(3, counters.NAccepted)
	assert.EqualValues(4, counters.NOutOfOrder)
	assert.EqualValues(1, counters.NDelivered)
}
