package pit_test

import (
	"fmt"
	"testing"

	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn/ndntestutil"
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

	interestAB := ndntestutil.MakeInterest("050E name=0706 080141 080142 nonce=0A04A0A1A2A3")
	defer ndntestutil.ClosePacket(interestAB)

	pitEntryAB, csEntryAB := pit.Insert(interestAB)
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
	tokens, entries := make([]uint64, 255), make([]pit.Entry, 255)

	pit := createPit()
	defer pit.Pcct.Close()
	defer pit.Close()

	for i := 0; i <= 255; i++ {
		interest := ndntestutil.MakeInterest(fmt.Sprintf("/I/%d", i))

		entry, _ := pit.Insert(interest)
		if i == 255 { // PCCT is full
			assert.Nil(entry)
			ndntestutil.ClosePacket(interest)
			continue
		}
		require.NotNil(entry)
		entry.DnRxInterest(interest)

		token := entry.GetToken()
		assert.Equal(token&(1<<48-1), token) // token has 48 bits
		tokens[i] = token
		entries[i] = *entry
	}

	assert.Equal(255, pit.Len())
	assert.Len(tokens, 255)

	for i, token := range tokens {
		entry := entries[i]
		data := ndntestutil.MakeData(fmt.Sprintf("/I/%d", i))
		defer ndntestutil.ClosePacket(data)
		ndntestutil.SetPitToken(data, token)
		found := pit.FindByData(data)
		if assert.Len(found, 1) {
			assert.True(entry.SameAs(*found[0]))
		}

		// high 16 bits of the token should be ignored
		token2 := token ^ 0x79BC000000000000
		ndntestutil.SetPitToken(data, token2)
		found = pit.FindByData(data)
		if assert.Len(found, 1) {
			assert.True(entry.SameAs(*found[0]))
		}

		// name mismatch
		data2 := ndntestutil.MakeData(fmt.Sprintf("/K/%d", i))
		defer ndntestutil.ClosePacket(data2)
		ndntestutil.SetPitToken(data2, token)
		found = pit.FindByData(data2)
		assert.Len(found, 0)

		pit.Erase(entry)
		found = pit.FindByData(data)
		assert.Len(found, 0)
	}

	cnt := pit.ReadCounters()
	assert.Equal(uint64(255), cnt.NInsert)
	assert.Equal(uint64(1), cnt.NAllocErr)
	assert.Equal(uint64(510), cnt.NHits)
	assert.Equal(uint64(510), cnt.NMisses)
}
