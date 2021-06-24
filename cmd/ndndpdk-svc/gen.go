package main

import (
	"github.com/usnistgov/ndn-dpdk/app/tg"
)

type genArgs struct {
	CommonArgs
}

func (a genArgs) Activate() error {
	if e := a.CommonArgs.apply(); e != nil {
		return e
	}

	tg.GqlEnabled = true
	return nil
}
