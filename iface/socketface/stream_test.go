package socketface_test

import (
	"io"
	"net"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/ndn"
)

func TestStream(t *testing.T) {
	assert, _ := makeAR(t)

	conn1, conn2 := net.Pipe()

	face1 := socketface.New(conn1, socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	defer face1.Close()

	hexPkts := []string{
		"interest 050B name=0703080141 nonce=0A04CACBCCCD",
		"data 0609 name=0703080141 meta=1400 content=1500",
		"nack 6418 nack=FD032005(FD03210196~noroute) " +
			"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)",
	}
	txPkts := make([]ndn.Packet, 3)
	var hexJoined []byte
	for i, hexPkt := range hexPkts {
		hexJoined = append(hexJoined, dpdktestenv.PacketBytesFromHex(hexPkt)...)

		txPkts[i] = ndn.PacketFromDpdk(dpdktestenv.PacketFromHex(hexPkt))
		defer txPkts[i].AsDpdkPacket().Close()
	}

	go conn2.Write(hexJoined)
	time.Sleep(time.Millisecond * 100)
	rxPkts := make([]ndn.Packet, 3)
	nRxPkts := face1.RxBurst(rxPkts)
	assert.Equal(3, nRxPkts)

	txDone := false
	go func() {
		readBuf := make([]byte, len(hexJoined))
		n, e := io.ReadFull(conn2, readBuf)
		assert.NoError(e)
		assert.Equal(len(hexJoined), n)
		txDone = true
	}()
	face1.TxBurst(txPkts)
	time.Sleep(time.Millisecond * 100)
	assert.True(txDone)

	cnt := face1.ReadCounters()
	assert.EqualValues(3, cnt.RxL2.NFrames)
	assert.EqualValues(1, cnt.RxL3.NInterests)
	assert.EqualValues(1, cnt.RxL3.NData)
	assert.EqualValues(1, cnt.RxL3.NNacks)
	assert.EqualValues(3, cnt.TxL2.NFrames)
	// TxL3 counters are unavailable because packets do not have L3PktType specified.
}
