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

	entry4 := fixture.Insert(ndntestutil.MakeInterest("/A/2",
		ndn.FHDelegation{1, "/F"}, ndn.FHDelegation{1, "/G"}, ndn.ActiveFHDelegation(0)))
	assert.NotNil(entry4)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry4.GetPtr()))

	entry5 := fixture.Insert(ndntestutil.MakeInterest("/A/2",
		ndn.FHDelegation{1, "/F"}, ndn.FHDelegation{1, "/G"}, ndn.ActiveFHDelegation(1)))
	assert.NotNil(entry5)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry5.GetPtr()))
	assert.NotEqual(uintptr(entry4.GetPtr()), uintptr(entry5.GetPtr()))

	interest6 := ndntestutil.MakeInterest("/A/2", ndn.MustBeFreshFlag)
	entry6 := fixture.Insert(interest6)
	assert.NotNil(entry6)
	assert.NotEqual(uintptr(entry2.GetPtr()), uintptr(entry6.GetPtr()))

	assert.Equal(5, fixture.Pit.Len())
	assert.Equal(5, fixture.CountMpInUse()) // entry2 and entry6 share a PccEntry but it has PccEntryExt

	fixture.Pit.Erase(*entry6) // entry6 is on PccEntryExt, removing it should release PccEntryExt
	assert.Equal(4, fixture.Pit.Len())
	assert.Equal(4, fixture.CountMpInUse())

	fixture.Pit.Erase(*entry1)
	fixture.Pit.Erase(*entry2)
	fixture.Pit.Erase(*entry4)
	fixture.Pit.Erase(*entry5)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestToken(t *testing.T) {
	assert, require := makeAR(t)
	interestNames := make([]string, 255)
	dataPkts := make([]*ndn.Data, 255)
	entries := make([]pit.Entry, 255)
	fixture := NewFixture(255)
	defer fixture.Close()
	pit := fixture.Pit

	for i := 0; i <= 255; i++ {
		data := ndntestutil.MakeData(fmt.Sprintf("/I/%d", i))
		name := data.GetName().String()
		if i < 32 {
			name = data.GetFullName().String()
		}
		interest := ndntestutil.MakeInterest(name)

		entry, _ := pit.Insert(interest, fixture.EmptyFibEntry)
		if i == 255 { // PCCT is full
			assert.Nil(entry)
			ndntestutil.ClosePacket(data)
			ndntestutil.ClosePacket(interest)
			continue
		}
		require.NotNil(entry, "unexpected PCCT full at %d", i)

		token := entry.GetToken()
		assert.Equal(token&(1<<48-1), token) // token has 48 bits
		ndntestutil.SetPitToken(data, token)

		interestNames[i] = name
		dataPkts[i] = data
		entries[i] = *entry
	}

	assert.Equal(255, pit.Len())
	assert.Len(entries, 255)

	for i, entry := range entries {
		name := interestNames[i]
		data := dataPkts[i]
		token := ndntestutil.GetPitToken(data)

		found := pit.FindByData(data)
		foundEntries := found.ListEntries()
		if assert.Len(foundEntries, 1) {
			assert.Equal(uintptr(entry.GetPtr()), uintptr(foundEntries[0].GetPtr()))
		}

		// Interest carries implicit digest, so Data digest is needed
		if i < 32 && assert.True(found.NeedDataDigest()) {
			data.ComputeDigest(true)
			found = pit.FindByData(data)
			foundEntries = found.ListEntries()
			if assert.Len(foundEntries, 1) {
				assert.Equal(uintptr(entry.GetPtr()), uintptr(foundEntries[0].GetPtr()))
			}
		}
		assert.False(found.NeedDataDigest())

		// high 16 bits of the token should be ignored
		token2 := token ^ 0x79BC000000000000
		nack := ndn.MakeNackFromInterest(ndntestutil.MakeInterest(name),
			ndn.NackReason_NoRoute)
		ndntestutil.SetPitToken(nack, token2)
		foundEntry := pit.FindByNack(nack)
		if assert.NotNil(foundEntry) {
			assert.Equal(uintptr(entry.GetPtr()), uintptr(foundEntry.GetPtr()))
		}

		// name mismatch
		data2 := ndntestutil.MakeData(fmt.Sprintf("/K/%d", i))
		ndntestutil.SetPitToken(data2, token)
		foundEntries = pit.FindByData(data2).ListEntries()
		assert.Len(foundEntries, 0)

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
