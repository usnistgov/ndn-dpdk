package fwdpmgmt

type IndexArg struct {
	Index int
}

type FwdpInfo struct {
	NInputs int
	NFwds   int
}

type CsCounters struct {
	Capacity int
	NEntries int
	NHits    uint64
	NMisses  uint64
}
