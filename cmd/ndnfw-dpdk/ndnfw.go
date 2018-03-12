package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/app/fwdp/fwdpmgmt"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

var theNdt ndt.Ndt
var theFib *fib.Fib
var theDp *fwdp.DataPlane

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	appinit.InitEal()
	startDp()
	appinit.EnableMgmt()
	fwdpmgmt.Enable(theDp)
	appinit.StartMgmt()

	// set FIB nexthops
	// TODO remove this when FIB management is ready
	{
		var fibEntry fib.Entry
		fibEntryName, _ := ndn.ParseName("/")
		fibEntry.SetName(fibEntryName)
		fibNextHops := make([]iface.FaceId, 0)
		for _, face := range appinit.GetFaceTable().ListFaces() {
			fibNextHops = append(fibNextHops, face.GetFaceId())
			if len(fibNextHops) >= fib.MAX_NEXTHOPS {
				break
			}
		}
		fibEntry.SetNexthops(fibNextHops)
		theFib.Insert(&fibEntry)
	}

	select {}
}

func startDp() {
	logger := log.New(os.Stderr, "startDp ", log.LstdFlags)
	logger.Printf("EAL has %d slave lcores", len(appinit.Eal.Slaves))
	lcr := appinit.NewLCoreReservations()

	var dpCfg fwdp.Config
	dpCfg.FaceTable = appinit.GetFaceTable()

	// reserve lcores for EthFace inputs
	var inputNumaSockets []dpdk.NumaSocket
	var inputRxLoopers []iface.IRxLooper
	for _, port := range dpdk.ListEthDevs() {
		face, e := appinit.NewFaceFromEthDev(port)
		if e != nil {
			logger.Printf("%v", e)
			continue
		}
		lc := lcr.ReserveRequired(face.GetNumaSocket())
		socket := lc.GetNumaSocket()
		logger.Printf("Reserving lcore %d on socket %d for EthDev %d", lc, socket, port)
		dpCfg.InputLCores = append(dpCfg.InputLCores, lc)
		inputNumaSockets = append(inputNumaSockets, socket)
		inputRxLoopers = append(inputRxLoopers, appinit.MakeRxLooper(*face))
	}

	// TODO reserve lcore for SocketFace inputs

	// initialize NDT
	{
		var ndtCfg ndt.Config
		ndtCfg.PrefixLen = 2
		ndtCfg.IndexBits = 16
		ndtCfg.SampleFreq = 8
		theNdt = ndt.New(ndtCfg, inputNumaSockets)
		dpCfg.Ndt = theNdt
	}

	// initialize FIB
	{
		var fibCfg fib.Config
		fibCfg.Id = "FIB"
		fibCfg.MaxEntries = 65535
		fibCfg.NBuckets = 256
		fibCfg.NumaSocket = dpdk.GetMasterLCore().GetNumaSocket()
		fibCfg.StartDepth = 8
		var e error
		theFib, e = fib.New(fibCfg)
		if e != nil {
			appinit.Exitf(appinit.EXIT_MEMPOOL_INIT_ERROR, "fib.New(): %v", e)
		}
		dpCfg.Fib = theFib
	}

	// reserve lcores for forwarding processes
	nFwds := len(appinit.Eal.Slaves) - len(dpCfg.InputLCores)
	for len(dpCfg.FwdLCores) < nFwds {
		lc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
		if !lc.IsValid() {
			break
		}
		logger.Printf("Reserving lcore %d on socket %d for forwarding", lc, lc.GetNumaSocket())
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}

	// set forwarding process config
	dpCfg.FwdQueueCapacity = 64
	dpCfg.PcctCfg.MaxEntries = 65535

	// create dataplane
	{
		var e error
		theDp, e = fwdp.New(dpCfg)
		if e != nil {
			appinit.Exitf(appinit.EXIT_EAL_LAUNCH_ERROR, "fwdp.New(): %v", e)
		}
	}

	// launch forwarding lcores
	logger.Print("Launching forwarding lcores")
	for i := range dpCfg.FwdLCores {
		e := theDp.LaunchFwd(i)
		if e != nil {
			appinit.Exitf(appinit.EXIT_EAL_LAUNCH_ERROR, "dp.LaunchFwd(%d): %v", i, e)
		}
	}

	// launch input lcores
	logger.Print("Launching input lcores")
	const burstSize = 64
	for i, rxl := range inputRxLoopers {
		e := theDp.LaunchInput(i, rxl, burstSize)
		if e != nil {
			appinit.Exitf(appinit.EXIT_EAL_LAUNCH_ERROR, "dp.LaunchInput(%d): %v", i, e)
		}
	}

	// randomize NDT
	theNdt.Randomize(nFwds)

	logger.Print("Data plane started")
}
