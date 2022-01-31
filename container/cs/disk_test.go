package cs_test

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
)

func TestDisk(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{
		CsDirectCapacity: 200,
	})
	fixture.EnableDisk(500)

	for i := 1; i < 600; i++ {
		fixture.Insert(makeInterest(fmt.Sprintf("/N/%d", i)), makeData(fmt.Sprintf("/N/%d", i)))
		fixture.Find(makeInterest(fmt.Sprintf("/N/%d", i)))
	}
	assert.Equal(200, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(200, fixture.Cs.CountEntries(cs.ListDirectB2))

	cnt := fixture.Cs.Counters()
	assert.EqualValues(200, cnt.NDiskInsert-cnt.NDiskDelete)
	assert.NotZero(cnt.NDiskInsert)
	assert.NotZero(cnt.NDiskDelete)
}
