package fwdpmgmt

type IndexArg struct {
	Index int
}

type FwdpInfo struct {
	NInputs int
	NFwds   int
}

type CsListCounters struct {
	Count    int
	Capacity int
}

type CsCounters struct {
	MD CsListCounters // in-memory direct entries
	MI CsListCounters // in-memory indirect entries

	NHits   uint64
	NMisses uint64
}
