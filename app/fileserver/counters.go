package fileserver

// Counters contains file server counters.
type Counters struct {
	ReqRead             uint64 `json:"reqRead" gqldesc:"Received read requests."`
	ReqLs               uint64 `json:"reqLs" gqldesc:"Received directory listing requests."`
	ReqMetadata         uint64 `json:"reqMetadata" gqldesc:"Received metadata requests."`
	FdNew               uint64 `json:"fdNew" gqldesc:"Successfully opened file descriptors."`
	FdNotFound          uint64 `json:"fdNotFound" gqldesc:"File not found."`
	FdUpdateStat        uint64 `json:"fdUpdateStat" gqldesc:"Update stat on already open file descriptors."`
	FdClose             uint64 `json:"fdClose" gqldesc:"Closed file descriptors."`
	UringSubmit         uint64 `json:"uringSubmit" gqldesc:"uring submissions total."`
	UringSubmitNonBlock uint64 `json:"uringSubmitNonBlock" gqldesc:"uring submissions non-blocking."`
	UringSubmitWait     uint64 `json:"uringSubmitWait" gqldesc:"uring submissions waiting for completions."`
	SqeSubmit           uint64 `json:"sqeSubmit" gqldesc:"I/O submissions total."`
	CqeFail             uint64 `json:"cqeFail" gqldesc:"I/O completions with errors."`
}

func (cnt *Counters) add(c countersC) {
	cnt.ReqRead += c.ReqRead
	cnt.ReqLs += c.ReqLs
	cnt.ReqMetadata += c.ReqMetadata
	cnt.FdNew += c.FdNew
	cnt.FdNotFound += c.FdNotFound
	cnt.FdUpdateStat += c.FdUpdateStat
	cnt.FdClose += c.FdClose
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
