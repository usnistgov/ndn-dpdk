package main

import (
	"strings"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestParseCommand(t *testing.T) {
	assert, _ := makeAR(t)

	pc, e := parseCommand(strings.Split("-rtt +c dev://net_pcap0 /prefix/ping 100", " "))
	if assert.NoError(e) {
		// TODO verify client config
	}

	pc, e = parseCommand(strings.Split("-nack=false +s dev://net_pcap0 /P/ping /Q", " "))
	if assert.NoError(e) {
		assert.False(pc.serverNack)
		if assert.Len(pc.servers, 1) {
			assert.Equal("dev://net_pcap0", pc.servers[0].face.String())
			if assert.Len(pc.servers[0].prefixes, 2) {
				assert.EqualValues(dpdktestenv.PacketBytesFromHex("080150 080470696E67"),
					pc.servers[0].prefixes[0])
				assert.EqualValues(dpdktestenv.PacketBytesFromHex("080151"),
					pc.servers[0].prefixes[1])
			}
		}
	}
}
