package main

import (
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/hrlog"
	"github.com/usnistgov/ndn-dpdk/app/pdump"
	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"go.uber.org/zap"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/memifface"
)

// CommonArgs contains arguments shared between forwarder and traffic generator.
type CommonArgs struct {
	Eal        ealconfig.Config        `json:"eal,omitempty"`
	Mempool    pktmbuf.TemplateUpdates `json:"mempool,omitempty"`
	LCoreAlloc ealthread.Config        `json:"lcoreAlloc,omitempty"`
	SocketFace socketface.GlobalConfig `json:"socketFace,omitempty"`
}

func (a *CommonArgs) apply() error {
	args, e := a.Eal.Args(nil)
	if e != nil {
		return e
	}
	if e := ealinit.Init(args); e != nil {
		return e
	}

	a.Mempool.Apply()

	lcoreAlloc, e := a.LCoreAlloc.Extract(map[string]int{hrlog.Role: 0, pdump.Role: 0})
	if e != nil {
		return e
	}
	alloc, e := ealthread.AllocConfig(lcoreAlloc)
	if e != nil {
		return e
	}
	if lc := alloc[hrlog.Role]; len(lc) > 0 {
		hrlog.GqlLCore = lc[0]
	}
	if lc := alloc[pdump.Role]; len(lc) > 0 {
		pdump.GqlLCore = lc[0]
	}

	a.SocketFace.Apply()
	return nil
}

func initXDPProgram() {
	path, e := bpf.XDP.Find("redir")
	if e != nil {
		logger.Warn("XDP program not found, AF_XDP may not work correctly", zap.Error(e))
		return
	}

	ethnetif.XDPProgram = path
}

var shutdownOnce sync.Once

func delayedShutdown(then func()) {
	// Shutdown is slightly delayed to allow enough time to send back the GraphQL result.
	// It's possible to receive shutdown command from both GraphQL and os.Signal at the same time,
	// so that the cleanup step must be protected by sync.Once.

	go func() {
		shutdownOnce.Do(func() {
			iface.CloseAll()
		})
		time.Sleep(100 * time.Millisecond)
		then()
		logger.Panic("delayedShutdown then() must not return")
	}()
}
