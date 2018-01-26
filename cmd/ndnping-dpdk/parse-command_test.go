package main

import (
	"strings"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestParseCommand(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	pc, e := parseCommand(strings.Split(
		"-rtt -nack=false +c dev://net_pcap1 1ms /P/ping 60 /Q 70 +s dev://net_pcap0 /P/ping /Q", " "))
	if assert.NoError(e) {
		assert.False(pc.measureLatency)
		assert.True(pc.measureRtt)
		if assert.Len(pc.clients, 1) {
			assert.Equal("dev://net_pcap1", pc.clients[0].face.String())
			assert.Equal(time.Millisecond, pc.clients[0].interval)
			if assert.Len(pc.clients[0].patterns, 2) {
				assert.EqualValues(dpdktestenv.PacketBytesFromHex("080150 080470696E67"),
					pc.clients[0].patterns[0].prefix)
				assert.EqualValues(60.0, pc.clients[0].patterns[0].pct)
				assert.EqualValues(dpdktestenv.PacketBytesFromHex("080151"),
					pc.clients[0].patterns[1].prefix)
				assert.EqualValues(70.0, pc.clients[0].patterns[1].pct)
			}
		}

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
