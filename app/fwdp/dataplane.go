// Package fwdp implements the forwarder's data plane.
package fwdp

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

var logger = logging.New("fwdp")

// Thread roles.
const (
	RoleInput  = iface.RoleRx
	RoleOutput = iface.RoleTx
	RoleCrypto = "CRYPTO"
	RoleDisk   = "DISK"
	RoleFwd    = "FWD"
)

// Config contains data plane configuration.
type Config struct {
	LCoreAlloc ealthread.Config `json:"-"`

	Ndt      ndt.Config         `json:"ndt,omitempty"`
	Fib      fibdef.Config      `json:"fib,omitempty"`
	Pcct     pcct.Config        `json:"pcct,omitempty"`
	Suppress pit.SuppressConfig `json:"suppress,omitempty"`

	Crypto                CryptoConfig         `json:"crypto,omitempty"`
	Disk                  DiskConfig           `json:"disk,omitempty"`
	FwdInterestQueue      iface.PktQueueConfig `json:"fwdInterestQueue,omitempty"`
	FwdDataQueue          iface.PktQueueConfig `json:"fwdDataQueue,omitempty"`
	FwdNackQueue          iface.PktQueueConfig `json:"fwdNackQueue,omitempty"`
	LatencySampleInterval int                  `json:"latencySampleInterval,omitempty"`
}

func (cfg *Config) validate() error {
	if len(cfg.LCoreAlloc) > 0 {
		if e := cfg.LCoreAlloc.ValidateRoles(map[string]int{RoleInput: 1, RoleOutput: 1, RoleCrypto: 0, RoleDisk: 0, RoleFwd: 1}); e != nil {
			return e
		}
	}

	if cfg.FwdDataQueue.DequeueBurstSize <= 0 {
		cfg.FwdDataQueue.DequeueBurstSize = iface.MaxBurstSize
	}
	if cfg.FwdNackQueue.DequeueBurstSize <= 0 {
		cfg.FwdNackQueue.DequeueBurstSize = cfg.FwdDataQueue.DequeueBurstSize
	}
	if cfg.FwdInterestQueue.DequeueBurstSize <= 0 {
		cfg.FwdInterestQueue.DequeueBurstSize = max(cfg.FwdDataQueue.DequeueBurstSize/2, 1)
	}
	if cfg.LatencySampleInterval <= 0 {
		cfg.LatencySampleInterval = 1 << 16
	}
	return nil
}

// DefaultAlloc is the default lcore allocation algorithm.
func DefaultAlloc() (m map[string]eal.LCores, e error) {
	m = map[string]eal.LCores{}
	tryAlloc := func(reqs []ealthread.AllocReq) error {
		lc, e := ealthread.AllocRequest(reqs...)
		if e != nil {
			return e
		}
		for i, req := range reqs {
			m[req.Role] = append(m[req.Role], lc[i])
		}
		return nil
	}

	reqs := []ealthread.AllocReq{{Role: RoleCrypto}}
	for _, socket := range eal.Sockets {
		reqs = append(reqs,
			ealthread.AllocReq{Role: RoleFwd},
			ealthread.AllocReq{Role: RoleInput, Socket: socket},
			ealthread.AllocReq{Role: RoleOutput, Socket: socket},
		)
	}

	if tryAlloc(reqs) != nil {
		reqs = reqs[:4]
		for i := range reqs {
			reqs[i].Socket = eal.NumaSocket{}
		}
		if e := tryAlloc(reqs); e != nil {
			return nil, e
		}
	}

	reqs = []ealthread.AllocReq{{Role: RoleFwd}, {Role: RoleOutput}, {Role: RoleInput}, {Role: RoleFwd}}
	for {
		if tryAlloc(reqs) == nil {
			continue
		}
		if len(reqs) == 1 {
			break
		}
		reqs = reqs[1:]
	}

	return m, nil
}

// DataPlane represents the forwarder data plane.
type DataPlane struct {
	ndt      *ndt.Ndt
	fib      *fib.Fib
	dispatch []DispatchThread
	fwis     []*Input
	fwcs     []*Crypto
	fwcsh    map[eal.NumaSocket]*CryptoShared
	fwdisk   *Disk
	fwds     []*Fwd
}

// Ndt returns the NDT.
func (dp *DataPlane) Ndt() *ndt.Ndt {
	return dp.ndt
}

// Fib returns the FIB.
func (dp *DataPlane) Fib() *fib.Fib {
	return dp.fib
}

// Fwds returns a list of forwarding threads.
func (dp *DataPlane) Fwds() []*Fwd {
	return dp.fwds
}

// Close stops the data plane and releases resources.
func (dp *DataPlane) Close() error {
	var lcores eal.LCores
	deferFreeLCore := func(lc eal.LCore) {
		if lc.Valid() {
			lcores = append(lcores, lc)
		}
	}
	errs := []error{}

	for _, rxl := range iface.ListRxLoops() {
		lcores = append(lcores, rxl.LCore())
	}
	for _, txl := range iface.ListTxLoops() {
		lcores = append(lcores, txl.LCore())
	}

	errs = append(errs, iface.CloseAll())
	if dp.ndt != nil {
		errs = append(errs, dp.ndt.Close())
	}
	for _, fwc := range dp.fwcs {
		deferFreeLCore(fwc.LCore())
		errs = append(errs, fwc.Close())
	}
	for _, fwcsh := range dp.fwcsh {
		errs = append(errs, fwcsh.Close())
	}
	if dp.fwdisk != nil {
		deferFreeLCore(dp.fwdisk.LCore())
		errs = append(errs, dp.fwdisk.Close())
	}
	for _, fwd := range dp.fwds {
		deferFreeLCore(fwd.LCore())
		errs = append(errs, fwd.Close())
	}
	for _, fwi := range dp.fwis {
		errs = append(errs, fwi.Close())
	}
	if dp.fib != nil {
		errs = append(errs, dp.fib.Close())
	}

	ealthread.AllocFree(lcores...)
	return errors.Join(errs...)
}

// New creates and launches forwarder data plane.
func New(cfg Config) (dp *DataPlane, e error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}
	dp = &DataPlane{}
	defer func(d *DataPlane) {
		if e != nil {
			must.Close(d)
		}
	}(dp)

	var alloc map[string]eal.LCores
	if len(cfg.LCoreAlloc) > 0 {
		alloc, e = ealthread.AllocConfig(cfg.LCoreAlloc)
	} else {
		alloc, e = DefaultAlloc()
	}
	if e != nil {
		return nil, e
	}
	lcRx, lcTx, lcCrypto, lcDisk, lcFwd := alloc[RoleInput], alloc[RoleOutput], alloc[RoleCrypto], alloc[RoleDisk], alloc[RoleFwd]

	{
		ndtSockets := []eal.NumaSocket{}
		for _, lcs := range []eal.LCores{lcRx, lcDisk} {
			for socket := range lcs.ByNumaSocket() {
				ndtSockets = append(ndtSockets, socket)
			}
		}
		dp.ndt = ndt.New(cfg.Ndt, ndtSockets)
		dp.ndt.Randomize(uint8(len(lcFwd)))
	}

	for _, lc := range lcTx {
		txl := iface.NewTxLoop(lc.NumaSocket())
		txl.SetLCore(lc)
		ealthread.Launch(txl)
	}

	var fibFwds []fib.LookupThread
	for i, lc := range lcFwd {
		fwd, e := newFwd(i, lc, cfg.Pcct, cfg.FwdInterestQueue, cfg.FwdDataQueue, cfg.FwdNackQueue,
			cfg.LatencySampleInterval, cfg.Suppress)
		if e != nil {
			return nil, fmt.Errorf("Fwd[%d].Init(): %w", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
		fibFwds = append(fibFwds, fwd)
	}
	if len(eal.Sockets)*ndni.PacketMempool.Config().Capacity < len(dp.fwds)*cfg.Pcct.CsMemoryCapacity {
		logger.Warn("total DIRECT mempool capacity is less than total CsMemoryCapacity; packet reception will stop when CS is full")
	}

	if dp.fib, e = fib.New(cfg.Fib, fibFwds); e != nil {
		return nil, fmt.Errorf("fib.New: %w", e)
	}

	demuxPrep := &demuxPreparer{
		Ndt:  dp.ndt,
		Fwds: dp.fwds,
	}

	fwcshList := []*CryptoShared{}
	{
		dp.fwcsh = map[eal.NumaSocket]*CryptoShared{}
		for socket, lcs := range lcCrypto.ByNumaSocket() {
			socketFwcs := []*Crypto{}
			for i, lc := range lcs {
				if _, e := addDispatchThread(dp, &socketFwcs, func(id int) (*Crypto, error) {
					return newCrypto(id, lc, demuxPrep)
				}); e != nil {
					return nil, fmt.Errorf("Crypto[%d].Init(): %w", i, e)
				}
			}

			fwcsh, e := newCryptoShared(cfg.Crypto, socket, len(socketFwcs))
			if e != nil {
				return nil, fmt.Errorf("newCryptoShared[%s]: %w", socket, e)
			}
			fwcsh.AssignTo(socketFwcs)
			fwcshList = append(fwcshList, fwcsh)
			dp.fwcsh[socket] = fwcsh
			dp.fwcs = append(dp.fwcs, socketFwcs...)
		}
	}

	if len(lcDisk) > 0 {
		cfg.Disk.csDiskCapacity = cfg.Pcct.CsDiskCapacity
		dp.fwdisk, e = addDispatchThread(dp, nil, func(id int) (*Disk, error) {
			return newDisk(id, lcDisk[0], demuxPrep, cfg.Disk)
		})
		if e != nil {
			return nil, fmt.Errorf("Disk[%d].Init(): %w", 0, e)
		}
	} else if cfg.Pcct.CsDiskCapacity > 0 {
		logger.Warn("CsDiskCapacity is non-zero but no lcore is allocated for DISK role; disk caching will not work")
	}

	for _, fwc := range dp.fwcs {
		ealthread.Launch(fwc)
	}
	if dp.fwdisk != nil {
		ealthread.Launch(dp.fwdisk)
	}
	for _, fwd := range dp.fwds {
		if fwcsh := dp.fwcsh[fwd.NumaSocket()]; fwcsh != nil {
			fwcsh.ConnectTo(fwd)
		} else if n := len(fwcshList); n > 0 {
			fwcshList[rand.Intn(n)].ConnectTo(fwd)
		}
		ealthread.Launch(fwd)
	}

	for i, lc := range lcRx {
		fwi, e := addDispatchThread(dp, &dp.fwis, func(id int) (*Input, error) {
			return newInput(id, lc, demuxPrep)
		})
		if e != nil {
			return nil, fmt.Errorf("Input[%d].Init(): %w", i, e)
		}
		ealthread.Launch(fwi.rxl)
	}

	iface.RxParseFor = ndni.ParseForFw
	return dp, nil
}

func addDispatchThread[T DispatchThread](dp *DataPlane, slice *[]T, f func(id int) (T, error)) (T, error) {
	id := len(dp.dispatch)
	th, e := f(id)
	if e == nil {
		dp.dispatch = append(dp.dispatch, th)
		if slice != nil {
			*slice = append(*slice, th)
		}
	}
	return th, e
}
