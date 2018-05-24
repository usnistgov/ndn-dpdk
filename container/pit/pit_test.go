package pit_test

import (
	"fmt"
	"testing"

	"ndn-dpdk/container/pit"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInsertErase(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())

	interest1 := ndntestutil.MakeInterest("/A/1")
	entry1 := fixture.Insert(interest1)
	assert.NotNil(entry1)

	interest2 := ndntestutil.MakeInterest("/A/2")
	entry2 := fixture.Insert(interest2)
	assert.NotNil(entry2)
	assert.NotEqual(uintptr(entry1.GetPtr()), uintptr(entry2.GetPtr()))

	interest3 := ndntestutil.MakeInterest("/A/2",
		ndn.FHDelegation{1, "/F"}, ndn.FHDelegation{1, "/G"})
	entry3 := fixture.Insert(interest3)
	ndntestutil.ClosePacket(interest3)
	assert.NotNil(entry3)
	assert.Equal(uintptr(entry2.GetPtr()), uintptr(entry3.GetPtr()))

	interest4 := ndntestutil.MakeInterest("/A/2",
		ndn.FHDelegation{1, "/F"}, ndn.FHDelegation{1, "/G"})
	interest4.SelectActiveFh(0)
	entry4 := fixture.Insert(interest4)
	assert.NotNil(entry4)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry4.GetPtr()))

	interest5 := ndntestutil.MakeInterest("/A/2",
		ndn.FHDelegation{1, "/F"}, ndn.FHDelegation{1, "/G"})
	interest5.SelectActiveFh(1)
	entry5 := fixture.Insert(interest5)
	assert.NotNil(entry5)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry5.GetPtr()))
	assert.NotEqual(uintptr(entry4.GetPtr()), uintptr(entry5.GetPtr()))

	interest6 := ndntestutil.MakeInterest("/A/2", ndn.MustBeFreshFlag)
	entry6 := fixture.Insert(interest6)
	assert.NotNil(entry6)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry6.GetPtr()))

	assert.Equal(5, fixture.Pit.Len())
	assert.Equal(4, fixture.CountMpInUse()) // entry2 and entry6 share a PccEntry

	fixture.Pit.Erase(*entry1)
	fixture.Pit.Erase(*entry2)
	fixture.Pit.Erase(*entry4)
	fixture.Pit.Erase(*entry5)
	fixture.Pit.Erase(*entry6)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestToken(t *testing.T) {
	assert, require := makeAR(t)
	tokens, entries := make([]uint64, 255), make([]pit.Entry, 255)
	fixture := NewFixture(255)
	defer fixture.Close()
	pit := fixture.Pit

	for i := 0; i <= 255; i++ {
		interest := ndntestutil.MakeInterest(fmt.Sprintf("/I/%d", i))

		entry, _ := pit.Insert(interest, fixture.EmptyFibEntry)
		if i == 255 { // PCCT is full
			assert.Nil(entry)
			ndntestutil.ClosePacket(interest)
			continue
		}
		require.NotNil(entry)

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
		ndntestutil.SetPitToken(data, token)
		found := pit.FindByData(data)
		if assert.Equal(1, found.Len()) {
			assert.Equal(uintptr(entry.GetPtr()), uintptr(found.GetEntries()[0].GetPtr()))
		}

		// high 16 bits of the token should be ignored
		token2 := token ^ 0x79BC000000000000
		nack := ndn.MakeNackFromInterest(ndntestutil.MakeInterest(fmt.Sprintf("/I/%d", i)),
			ndn.NackReason_NoRoute)
		ndntestutil.SetPitToken(nack, token2)
		foundEntry := pit.FindByNack(nack)
		if assert.NotNil(foundEntry) {
			assert.Equal(uintptr(entry.GetPtr()), uintptr(foundEntry.GetPtr()))
		}

		// name mismatch
		data2 := ndntestutil.MakeData(fmt.Sprintf("/K/%d", i))
		ndntestutil.SetPitToken(data2, token)
		found = pit.FindByData(data2)
		assert.Equal(0, found.Len())

		pit.Erase(entry)
		foundEntry = pit.FindByNack(nack)
		assert.Nil(foundEntry)

		ndntestutil.ClosePacket(data)
		ndntestutil.ClosePacket(nack)
		ndntestutil.ClosePacket(data2)
	}

	cnt := pit.ReadCounters()
	assert.Equal(uint64(255), cnt.NInsert)
	assert.Equal(uint64(1), cnt.NAllocErr)
	assert.Equal(uint64(255), cnt.NDataHit)
	assert.Equal(uint64(255), cnt.NDataMiss)
	assert.Equal(uint64(255), cnt.NNackHit)
	assert.Equal(uint64(255), cnt.NNackMiss)
}
