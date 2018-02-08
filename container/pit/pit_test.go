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
	d := ndn.NewTlvDecoder(pktAB)
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
