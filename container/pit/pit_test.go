package pit_test

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

func TestInsertErase(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, 255)

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
		ndn.ForwardingHint{ndn.ParseName("/F"), ndn.ParseName("/G")})
	entry3 := fixture.Insert(interest3)
	must.Close(interest3)
	assert.NotNil(entry3)
	assert.Equal(uintptr(entry2.Ptr()), uintptr(entry3.Ptr()))

	entry4 := fixture.Insert(makeInterest("/A/2",
		ndn.ForwardingHint{ndn.ParseName("/F"), ndn.ParseName("/G")}, setActiveFwHint(0)))
	assert.NotNil(entry4)
	assert.NotEqual(uintptr(entry2.Ptr()), uintptr(entry4.Ptr()))

	entry5 := fixture.Insert(makeInterest("/A/2",
		ndn.ForwardingHint{ndn.ParseName("/F"), ndn.ParseName("/G")}, setActiveFwHint(1)))
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

func TestCanBePrefixMustBeFresh(t *testing.T) {
	// insertTo shall insert two PIT entries at /A name, where CanBePrefix and MustBeFresh is each placed on only one
	// PIT entry, returns entry0 lacking CanBePrefix and entry1 having CanBePrefix
	test := func(insertTo func(fixture *Fixture) (entry0, entry1 *pit.Entry)) func(t *testing.T) {
		return func(t *testing.T) {
			fixture := NewFixture(t, 255)
			assert, _ := makeAR(t)

			entry0, entry1 := insertTo(fixture)
			assert.Equal(2, fixture.Pit.Len())
			assert.Equal(entry0.PitToken(), entry1.PitToken())

			if found := fixture.FindByData(makeData("/A/Z"), entry0.PitToken()).ListEntries(); assert.Len(found, 1) {
				assert.Equal(entry1.Ptr(), found[0].Ptr())
			}

			if found := fixture.FindByData(makeData("/A"), entry0.PitToken()).ListEntries(); assert.Len(found, 2) {
				if entry0.Ptr() == found[0].Ptr() {
					assert.Equal(entry0.Ptr(), found[0].Ptr())
					assert.Equal(entry1.Ptr(), found[1].Ptr())
				} else {
					assert.Equal(entry0.Ptr(), found[1].Ptr())
					assert.Equal(entry1.Ptr(), found[0].Ptr())
				}
			}
		}
	}

	t.Run("none => cbp+mbf", test(func(fixture *Fixture) (entry0 *pit.Entry, entry1 *pit.Entry) {
		entry0 = fixture.Insert(makeInterest("/A"))
		entry1 = fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag))
		return
	}))

	t.Run("cbp+mbf => none", test(func(fixture *Fixture) (entry0 *pit.Entry, entry1 *pit.Entry) {
		entry1 = fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag))
		entry0 = fixture.Insert(makeInterest("/A"))
		return
	}))

	t.Run("cbp => mbf", test(func(fixture *Fixture) (entry0 *pit.Entry, entry1 *pit.Entry) {
		entry1 = fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag))
		entry0 = fixture.Insert(makeInterest("/A", ndn.MustBeFreshFlag))
		return
	}))

	t.Run("mbf => cbp", test(func(fixture *Fixture) (entry0 *pit.Entry, entry1 *pit.Entry) {
		entry0 = fixture.Insert(makeInterest("/A", ndn.MustBeFreshFlag))
		entry1 = fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag))
		return
	}))
}

type testTokenRecord struct {
	name  string
	data  *ndni.Packet
	entry *pit.Entry
	token uint64
}

func TestToken(t *testing.T) {
	assert, require := makeAR(t)
	records := make([]testTokenRecord, 255)
	nImplicitDigest := 32
	nAllocErr := 2
	fixture := NewFixture(t, len(records))
	pit := fixture.Pit

	for i := 0; i < len(records)+nAllocErr; i++ {
		data := makeData(fmt.Sprintf("/I/%d", i))
		nData := data.ToNPacket().Data
		name := nData.Name.String()
		if i < nImplicitDigest {
			name = nData.FullName().String()
		}
		interest := makeInterest(name)

		entry, _ := pit.Insert(interest, fixture.FibEntry)
		if i >= len(records) { // PCCT is full
			assert.Nil(entry)
			must.Close(data)
			must.Close(interest)
			continue
		}
		require.NotNil(entry, "unexpected PCCT full at %d", i)

		token := entry.PitToken()
		assert.Less(token, uint64(1<<48))

		records[i] = testTokenRecord{
			name:  name,
			data:  data,
			entry: entry,
			token: token,
		}
	}
	assert.Len(records, pit.Len())

	for i, record := range records {
		found := pit.FindByData(record.data, record.token)
		foundEntries := found.ListEntries()
		if assert.Len(foundEntries, 1, i) {
			assert.Equal(uintptr(record.entry.Ptr()), uintptr(foundEntries[0].Ptr()), i)
		}

		// Interest carries implicit digest, so Data digest is needed
		if i < nImplicitDigest && assert.True(found.NeedDataDigest(), i) {
			record.data.ComputeDataImplicitDigest()
			found = pit.FindByData(record.data, record.token)
			foundEntries = found.ListEntries()
			if assert.Len(foundEntries, 1, i) {
				assert.Equal(uintptr(record.entry.Ptr()), uintptr(foundEntries[0].Ptr()), i)
			}
		}
		assert.False(found.NeedDataDigest(), i)

		// high 16 bits of the token should be ignored
		token2 := record.token ^ 0x79BC000000000000
		nack := makeNack(ndn.MakeInterest(record.name), an.NackNoRoute)
		foundEntry := pit.FindByNack(nack, token2)
		if assert.NotNil(foundEntry, i) {
			assert.Equal(uintptr(record.entry.Ptr()), uintptr(foundEntry.Ptr()), i)
		}

		// name mismatch
		data2 := makeData(fmt.Sprintf("/K/%d", i))
		foundEntries = pit.FindByData(data2, record.token).ListEntries()
		assert.Len(foundEntries, 0, i)

		pit.Erase(record.entry)
		foundEntry = pit.FindByNack(nack, record.token)
		assert.Nil(foundEntry, i)

		must.Close(record.data)
		must.Close(nack)
		must.Close(data2)
	}

	cnt := pit.Counters()
	assert.EqualValues(len(records), cnt.NInsert)
	assert.EqualValues(nAllocErr, cnt.NAllocErr)
	assert.EqualValues(len(records), cnt.NDataHit)
	assert.EqualValues(len(records), cnt.NDataMiss)
	assert.EqualValues(len(records), cnt.NNackHit)
	assert.EqualValues(len(records), cnt.NNackMiss)
}
