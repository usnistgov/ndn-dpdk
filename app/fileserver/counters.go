package fileserver

// Counters contains file server counters.
type Counters struct {
	ReqRead             uint64 `json:"reqRead"`             // received read requests
	ReqLs               uint64 `json:"reqLs"`               // received directory listing requests
	ReqMetadata         uint64 `json:"reqMetadata"`         // received metadata requests
	FdNew               uint64 `json:"fdNew"`               // successfully opened file descriptors
	FdNotFound          uint64 `json:"fdNotFound"`          // file not found
	FdUpdateStat        uint64 `json:"fdUpdateStat"`        // update stat on already open file descriptors
	UringSubmit         uint64 `json:"uringSubmit"`         // uring submissions total
	UringSubmitNonBlock uint64 `json:"uringSubmitNonBlock"` // uring submissions non-blocking
	UringSubmitWait     uint64 `json:"uringSubmitWait"`     // uring submissions waiting for completions
	SqeSubmit           uint64 `json:"sqeSubmit"`           // I/O submissions total
	CqeFail             uint64 `json:"cqeFail"`             // I/O completions with errors
}

func (cnt *Counters) add(c countersC) {
	cnt.ReqRead += c.ReqRead
	cnt.ReqLs += c.ReqLs
	cnt.ReqMetadata += c.ReqMetadata
	cnt.FdNew += c.FdNew
	cnt.FdNotFound += c.FdNotFound
	cnt.FdUpdateStat += c.FdUpdateStat
	cnt.UringSubmitNonBlock += c.UringSubmitNonBlock
	cnt.UringSubmitWait += c.UringSubmitWait
	cnt.SqeSubmit += c.SqeSubmit
	cnt.CqeFail += c.CqeFail

	cnt.UringSubmit = cnt.UringSubmitNonBlock + cnt.UringSubmitWait
}

// Counters retrieves counters.
func (p *Server) Counters() (cnt Counters) {
	for _, w := range p.workers {
		cnt.add(w.counters())
	}
	return cnt
}
