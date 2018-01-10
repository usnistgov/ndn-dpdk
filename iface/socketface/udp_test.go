package socketface

import (
	"net"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)

	addr1 := net.UDPAddr{net.ParseIP("127.0.0.1"), 7001, ""}
	addr2 := net.UDPAddr{net.ParseIP("127.0.0.1"), 7002, ""}
	conn1, e := net.DialUDP("udp", &addr2, &addr1)
	require.NoError(e)
	conn2, e := net.DialUDP("udp", &addr1, &addr2)
	require.NoError(e)

	face1 := New(conn1, Config{
		RxMp:         directMp,
		RxqCapacity:  64,
		TxIndirectMp: indirectMp,
		TxHeaderMp:   headerMp,
		TxqCapacity:  64,
	})
	defer face1.Close()

	hexPkts := []string{
		"interest 050B name=0703080141 nonce=0A04CACBCCCD",
		"data 0609 name=0703080141 meta=1400 content=1500",
		"nack 6418 nack=FD032005(FD03210196~noroute) " +
			"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)",
	}
	txPkts := make([]ndn.Packet, 3)
	for i, hexPkt := range hexPkts {
		conn2.Write(dpdktestenv.PacketBytesFromHex(hexPkt))

		txPkts[i] = ndn.Packet{dpdktestenv.PacketFromHex(hexPkt)}
		defer txPkts[i].Close()
	}

	time.Sleep(time.Millisecond * 100)
	rxPkts := make([]ndn.Packet, 3)
	nRxPkts := face1.RxBurst(rxPkts)
	assert.Equal(3, nRxPkts)

	face1.TxBurst(txPkts)
	conn2.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	readBuf := make([]byte, 2000)
	for _ = range hexPkts {
		_, e := conn2.Read(readBuf)
		assert.NoError(e)
	}

	cnt := face1.ReadCounters()
	assert.EqualValues(3, cnt.RxL2.NFrames)
	assert.EqualValues(1, cnt.RxL3.NInterests)
	assert.EqualValues(1, cnt.RxL3.NData)
	assert.EqualValues(1, cnt.RxL3.NNacks)
	assert.EqualValues(3, cnt.TxL2.NFrames)
	// TxL3 counters are unavailable because packets do not have NdnPktType specified.
}
