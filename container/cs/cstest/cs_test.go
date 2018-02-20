package cstest

import (
	"testing"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn/ndntestutil"
)

func createCs() cs.Cs {
	cfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: 255,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}

	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}
	return cs.Cs{pcct}
}

func pitFromCs(cs cs.Cs) pit.Pit {
	return pit.Pit{cs.Pcct}
}

func TestReplacePitEntry(t *testing.T) {
	assert, require := makeAR(t)

	cs := createCs()
	defer cs.Pcct.Close()
	defer cs.Close()
	pit := pitFromCs(cs)
	defer pit.Close()
	mp := cs.GetMempool()
	assert.Zero(pit.Len())
	assert.Zero(mp.CountInUse())

	interest := ndntestutil.MakeInterest("050E name=0706080141080142 nonce=0A04A0A1A2A3")
	defer ndntestutil.ClosePacket(interest)
	pitEntry, csEntry := pit.Insert(interest)
	assert.Nil(csEntry)
	require.NotNil(pitEntry)

	data := ndntestutil.MakeData("060E name=0706080141080142 metainfo=1400 content=1502C0C1")
	defer ndntestutil.ClosePacket(data)
	assert.Zero(cs.Len())
	cs.ReplacePitEntry(pitEntry, data)
	assert.Equal(1, cs.Len())
	assert.Zero(pit.Len())
	assert.Equal(1, mp.CountInUse())

	pitEntry, csEntry = pit.Insert(interest)
	assert.Nil(pitEntry)
	require.NotNil(csEntry)

	cs.Erase(*csEntry)
	assert.Zero(cs.Len())
	assert.Zero(mp.CountInUse())
}
