package main

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/app/tg"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

type genArgs struct {
	CommonArgs

	MinLCores int `json:"minLCores,omitempty"`
}

func (a genArgs) Activate() error {
	var req ealconfig.Request
	req.MinLCores = math.MaxInt(1, a.MinLCores)
	if e := a.CommonArgs.apply(req); e != nil {
		return e
	}

	tg.GqlEnabled = true
	return nil
}
