package pit_test

import (
	"testing"

	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func createPit() pit.Pit {
	cfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: 255,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}

	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}
	return pit.Pit{pcct}
}

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	pit := createPit()
	defer pit.Pcct.Close()
	defer pit.Close()
	mp := pit.GetMempool()
	assert.Zero(pit.Len())
	assert.Zero(mp.CountInUse())

	pktAB := dpdktestenv.PacketFromHex("050E name=0706 080141 080142 nonce=0A04A0A1A2A3")
	defer pktAB.Close()
	d := ndn.NewTlvDecodePos(pktAB)
	interestAB, e := d.ReadInterest()
	require.NoError(e)

	pitEntryAB, csEntryAB := pit.Insert(&interestAB)
	assert.Nil(csEntryAB)
	require.NotNil(pitEntryAB)

	assert.Equal(1, pit.Len())
	assert.Equal(1, mp.CountInUse())

	pit.Erase(*pitEntryAB)
	assert.Zero(pit.Len())
	assert.Zero(mp.CountInUse())
}

func TestToken(t *testing.T) {
	assert, require := makeAR(t)
	tokens := make(map[uint64]pit.Entry)

	pit := createPit()
	defer pit.Pcct.Close()
	defer pit.Close()

	pktBytes := dpdktestenv.PacketBytesFromHex("050B name=0703 080141 nonce=0A04A0A1A2A3")
	for i := 0; i <= 255; i++ {
		pktBytes[6] = byte(i)
		pkt := dpdktestenv.PacketFromBytes(pktBytes)
		defer pkt.Close()
		d := ndn.NewTlvDecodePos(pkt)
		interest, e := d.ReadInterest()
		require.NoError(e)

		entry, _ := pit.Insert(&interest)
		if i == 255 { // PCCT is full
			assert.Nil(entry)
			continue
		}

		require.NotNil(entry)
		token := pit.AddToken(*entry)
		assert.Equal(token&(1<<48-1), token) // token has 48 bits
		tokens[token] = *entry
	}

	assert.Equal(255, pit.Len())
	assert.Len(tokens, 255)

	for token, entry := range tokens {
		found := pit.Find(token)
		if assert.NotNil(found) {
			assert.True(entry.SameAs(*found))
		}

		// high 16 bits should be ignored
		token2 := token ^ 0x79BC000000000000
		found2 := pit.Find(token2)
		if assert.NotNil(found2) {
			assert.True(entry.SameAs(*found2))
		}

		pit.Erase(entry)
		found = pit.Find(token)
		assert.Nil(found)
		found2 = pit.Find(token2)
		assert.Nil(found2)
	}
}
