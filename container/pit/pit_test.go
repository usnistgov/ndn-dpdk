package pit_test

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestInsertErase(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())

	interest1 := makeInterest("/A/1")
	entry1 := fixture.Insert(interest1)
	assert.NotNil(entry1)

	interest2 := makeInterest("/A/2")
	entry2 := fixture.Insert(interest2)
	assert.NotNil(entry2)
	assert.NotEqual(uintptr(entry1.Ptr()), uintptr(entry2.Ptr()))

	interest3 := makeInterest("/A/2",
		ndn.MakeFHDelegation(1, "/F"), ndn.MakeFHDelegation(1, "/G"))
	entry3 := fixture.Insert(interest3)
	interest3.Close()
	assert.NotNil(entry3)
	assert.Equal(uintptr(entry2.Ptr()), uintptr(entry3.Ptr()))

	entry4 := fixture.Insert(makeInterest("/A/2",
		ndn.MakeFHDelegation(1, "/F"), ndn.MakeFHDelegation(1, "/G"), setActiveFwHint(0)))
	assert.NotNil(entry4)
	assert.NotEqual(uintptr(entry2.Ptr()), uintptr(entry4.Ptr()))

	entry5 := fixture.Insert(makeInterest("/A/2",
		ndn.MakeFHDelegation(1, "/F"), ndn.MakeFHDelegation(1, "/G"), setActiveFwHint(1)))
	assert.NotNil(entry5)
	assert.NotEqual(uintptr(entry2.Ptr()), uintptr(entry5.Ptr()))
	assert.NotEqual(uintptr(entry4.Ptr()), uintptr(entry5.Ptr()))

	interest6 := makeInterest("/A/2", ndn.MustBeFreshFlag)
	entry6 := fixture.Insert(interest6)
	assert.NotNil(entry6)
	assert.NotEqual(uintptr(entry2.Ptr()), uintptr(entry6.Ptr()))

	assert.Equal(5, fixture.Pit.Len())
	assert.Equal(5, fixture.CountMpInUse()) // entry2 and entry6 share a PccEntry but it has PccEntryExt

	fixture.Pit.Erase(entry6) // entry6 is on PccEntryExt, removing it should release PccEntryExt
	assert.Equal(4, fixture.Pit.Len())
	assert.Equal(4, fixture.CountMpInUse())

	fixture.Pit.Erase(entry1)
	fixture.Pit.Erase(entry2)
	fixture.Pit.Erase(entry4)
	fixture.Pit.Erase(entry5)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestToken(t *testing.T) {
	assert, require := makeAR(t)
	interestNames := make([]string, 255)
	dataPkts := make([]*ndni.Packet, 255)
	entries := make([]*pit.Entry, 255)
	fixture := NewFixture(255)
	defer fixture.Close()
	pit := fixture.Pit

	for i := 0; i <= 255; i++ {
		data := makeData(fmt.Sprintf("/I/%d", i))
		nData := data.ToNPacket().Data
		name := nData.Name.String()
		if i < 32 {
			name = nData.FullName().String()
		}
		interest := makeInterest(name)

		entry, _ := pit.Insert(interest, fixture.EmptyFibEntry)
		if i == 255 { // PCCT is full
			assert.Nil(entry)
			data.Close()
			interest.Close()
			continue
		}
		require.NotNil(entry, "unexpected PCCT full at %d", i)

		token := entry.PitToken()
		assert.Equal(token&(1<<48-1), token) // token has 48 bits
		data.SetPitToken(token)

		interestNames[i] = name
		dataPkts[i] = data
		entries[i] = entry
	}

	assert.Equal(255, pit.Len())
	assert.Len(entries, 255)

	for i, entry := range entries {
		name := interestNames[i]
		data := dataPkts[i]
		token := data.PitToken()

		found := pit.FindByData(data)
		foundEntries := found.ListEntries()
		if assert.Len(foundEntries, 1) {
			assert.Equal(uintptr(entry.Ptr()), uintptr(foundEntries[0].Ptr()))
		}

		// Interest carries implicit digest, so Data digest is needed
		if i < 32 && assert.True(found.NeedDataDigest()) {
			data.ComputeDataImplicitDigest()
			found = pit.FindByData(data)
			foundEntries = found.ListEntries()
			if assert.Len(foundEntries, 1) {
				assert.Equal(uintptr(entry.Ptr()), uintptr(foundEntries[0].Ptr()))
			}
		}
		assert.False(found.NeedDataDigest())

		// high 16 bits of the token should be ignored
		token2 := token ^ 0x79BC000000000000
		nack := makeNack(makeInterest(name, setPitToken(token2)), an.NackNoRoute)
		foundEntry := pit.FindByNack(nack)
		if assert.NotNil(foundEntry) {
			assert.Equal(uintptr(entry.Ptr()), uintptr(foundEntry.Ptr()))
		}

		// name mismatch
		data2 := makeData(fmt.Sprintf("/K/%d", i))
		data2.SetPitToken(token)
		foundEntries = pit.FindByData(data2).ListEntries()
		assert.Len(foundEntries, 0)

		pit.Erase(entry)
		foundEntry = pit.FindByNack(nack)
		assert.Nil(foundEntry)

		data.Close()
		nack.Close()
		data2.Close()
	}

	cnt := pit.ReadCounters()
	assert.Equal(uint64(255), cnt.NInsert)
	assert.Equal(uint64(1), cnt.NAllocErr)
	assert.Equal(uint64(255), cnt.NDataHit)
	assert.Equal(uint64(255), cnt.NDataMiss)
	assert.Equal(uint64(255), cnt.NNackHit)
	assert.Equal(uint64(255), cnt.NNackMiss)
}
