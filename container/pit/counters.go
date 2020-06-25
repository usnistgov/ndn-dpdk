package pit

import "fmt"

// Counters contains PIT counters.
type Counters struct {
	NEntries  uint64 // current number of entries
	NInsert   uint64 // how many inserts created a new PIT entry
	NFound    uint64 // how many inserts found an existing PIT entry
	NCsMatch  uint64 // how many inserts matched a CS entry
	NAllocErr uint64 // how many inserts failed due to allocation error
	NDataHit  uint64 // how many find-by-Data found PIT entry/entries
	NDataMiss uint64 // how many find-by-Data did not find PIT entry
	NNackHit  uint64 // how many find-by-Nack found PIT entry
	NNackMiss uint64 // how many find-by-Nack did not found PIT entry
	NExpired  uint64 // how many entries expired
}

func (cnt Counters) String() string {
	return fmt.Sprintf("%d entries, %d inserts, %d found, %d cs-match, %d alloc-err, "+
		"%d data-hit, %d data-miss, %d nack-hit, %d nack-miss, %d expired",
		cnt.NEntries, cnt.NInsert, cnt.NFound, cnt.NCsMatch, cnt.NAllocErr,
		cnt.NDataHit, cnt.NDataMiss, cnt.NNackHit, cnt.NNackMiss, cnt.NExpired)
}

// ReadCounters reads counters from this PIT.
func (pit *Pit) ReadCounters() (cnt Counters) {
	pitp := pit.getPriv()
	cnt.NEntries = uint64(pitp.nEntries)
	cnt.NInsert = uint64(pitp.nInsert)
	cnt.NFound = uint64(pitp.nFound)
	cnt.NCsMatch = uint64(pitp.nCsMatch)
	cnt.NAllocErr = uint64(pitp.nAllocErr)
	cnt.NDataHit = uint64(pitp.nDataHit)
	cnt.NDataMiss = uint64(pitp.nDataMiss)
	cnt.NNackHit = uint64(pitp.nNackHit)
	cnt.NNackMiss = uint64(pitp.nNackMiss)
	cnt.NExpired = uint64(pitp.timeoutSched.nTriggered)
	return cnt
}
