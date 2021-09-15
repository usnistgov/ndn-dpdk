package main

import (
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tg"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

type genArgs struct {
	CommonArgs
}

func (a genArgs) Activate() error {
	if e := a.CommonArgs.apply(); e != nil {
		return e
	}

	tg.GqlCreateEnabled = true
	return nil
}

type fileServerArgs struct {
	CommonArgs
	Face       iface.LocatorWrapper `json:"face"`
	FileServer fileserver.Config    `json:"fileServer"`
}

func (a fileServerArgs) Activate() error {
	if e := a.CommonArgs.apply(); e != nil {
		return e
	}

	var cfg tg.Config
	cfg.Face = a.Face
	cfg.FileServer = &a.FileServer
	gen, e := tg.New(cfg)
	if e != nil {
		return e
	}
	if e := gen.Launch(); e != nil {
		must.Close(gen)
		return e
	}

	return nil
}
