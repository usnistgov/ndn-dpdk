package main

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"time"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/mgmt/facemgmt"
	"ndn-dpdk/mgmt/fibmgmt"
	"ndn-dpdk/mgmt/fwdpmgmt"
	"ndn-dpdk/strategy/strategy_elf"
)

var theSocketRxg *socketface.RxGroup
var theSocketTxl *iface.MultiTxLoop
var theNdt ndt.Ndt
var theStrategy fib.StrategyCode
var theFib *fib.Fib
var theDp *fwdp.DataPlane

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	appinit.InitEal()
	startDp()
	theStrategy = loadStrategy("multicast")
	theStrategy.Ref()
	startMgmt()

	select {}
}

func startDp() {
	logger := log.New(os.Stderr, "startDp ", log.LstdFlags)
	logger.Printf("EAL has %d slave lcores", len(appinit.Eal.Slaves))
	lcr := appinit.NewLCoreReservations()

	var dpCfg fwdp.Config

	// reserve lcores for EthFace
	var inputNumaSockets []dpdk.NumaSocket
	var inputRxLoopers []iface.IRxLooper
	var outputLCores []dpdk.LCore
	var outputTxLoopers []iface.ITxLooper
	for _, port := range dpdk.ListEthDevs() {
		face, e := appinit.NewFaceFromEthDev(port)
		if e != nil {
			logger.Printf("%v", e)
			continue
		}
		inputLc := lcr.ReserveRequired(face.GetNumaSocket())
		socket := inputLc.GetNumaSocket()
		logger.Printf("Reserving lcore %d on socket %d for EthDev %d RX", inputLc, socket, port)
		dpCfg.InputLCores = append(dpCfg.InputLCores, inputLc)
		inputNumaSockets = append(inputNumaSockets, socket)
		inputRxLoopers = append(inputRxLoopers, appinit.MakeRxLooper(face))

		e = face.EnableThreadSafeTx(256)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "EthFace(%d).EnableThreadSafeTx(): %v",
				port, e)
		}

		outputLc := lcr.ReserveRequired(socket)
		logger.Printf("Reserving lcore %d on socket %d for EthDev %d TX", outputLc, socket, port)
		outputLCores = append(outputLCores, outputLc)
		outputTxLoopers = append(outputTxLoopers, appinit.MakeTxLooper(face))
	}

	// reserve lcore for SocketFaces
	{
		theSocketRxg = socketface.NewRxGroup()
		inputLc := lcr.ReserveRequired(dpdk.NUMA_SOCKET_ANY)
		socket := inputLc.GetNumaSocket()
		logger.Printf("Reserving lcore %d on socket %d for SocketFaces RX", inputLc, socket)
		dpCfg.InputLCores = append(dpCfg.InputLCores, inputLc)
		inputNumaSockets = append(inputNumaSockets, socket)
		inputRxLoopers = append(inputRxLoopers, theSocketRxg)

		theSocketTxl = iface.NewMultiTxLoop()
		outputLc := lcr.ReserveRequired(socket)
		logger.Printf("Reserving lcore %d on socket %d for SocketFaces TX", outputLc, socket)
		outputLCores = append(outputLCores, outputLc)
		outputTxLoopers = append(outputTxLoopers, theSocketTxl)
	}

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
	nFwds := len(appinit.Eal.Slaves) - len(dpCfg.InputLCores) - len(outputLCores)
	for len(dpCfg.FwdLCores) < nFwds {
		lc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
		if !lc.IsValid() {
			break
		}
		logger.Printf("Reserving lcore %d on socket %d for forwarding", lc, lc.GetNumaSocket())
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}
	nFwds = len(dpCfg.FwdLCores)
	if nFwds <= 0 {
		appinit.Exitf(appinit.EXIT_EAL_LAUNCH_ERROR, "No lcore available for forwarding")
	}

	// randomize NDT
	theNdt.Randomize(nFwds)

	// set forwarding process config
	dpCfg.FwdQueueCapacity = 64
	dpCfg.LatencySampleRate = 16
	dpCfg.PcctCfg.MaxEntries = 65535
	dpCfg.CsCapacity = 32768

	// create dataplane
	{
		var e error
		theDp, e = fwdp.New(dpCfg)
		if e != nil {
			appinit.Exitf(appinit.EXIT_EAL_LAUNCH_ERROR, "fwdp.New(): %v", e)
		}
	}

	// launch output lcores
	logger.Print("Launching output lcores")
	for i := range outputTxLoopers {
		func(i int) {
			outputLCores[i].RemoteLaunch(func() int {
				txl := outputTxLoopers[i]
				txl.TxLoop()
				return 0
			})
		}(i)
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

	logger.Print("Data plane started")
}

func createFace(u faceuri.FaceUri) (iface.FaceId, error) {
	if u.Scheme != "udp4" && u.Scheme != "tcp4" {
		return iface.FACEID_INVALID, errors.New("face creation only allows udp4 and tcp4 schemes")
	}

	face, e := appinit.NewFaceFromUri(u)
	if e != nil {
		return iface.FACEID_INVALID, e
	}

	face.EnableThreadSafeTx(64)
	theSocketRxg.AddFace(face.(*socketface.SocketFace))
	theSocketTxl.AddFace(face)
	return face.GetFaceId(), nil
}

func startMgmt() {
	facemgmt.CreateFace = createFace
	appinit.RegisterMgmt(facemgmt.FaceMgmt{})
	fibmgmt.TheStrategy = theStrategy
	appinit.RegisterMgmt(fibmgmt.FibMgmt{theFib})
	appinit.RegisterMgmt(fwdpmgmt.DpInfoMgmt{theDp})
	appinit.StartMgmt()
}

func loadStrategy(shortname string) fib.StrategyCode {
	elf, e := strategy_elf.Load(shortname)
	if e != nil {
		appinit.Exitf(appinit.EXIT_MGMT_ERROR, "strategy_elf.Load(%s): %v", shortname, e)
	}
	sc, e := theFib.LoadStrategyCode(elf)
	if e != nil {
		appinit.Exitf(appinit.EXIT_MGMT_ERROR, "fib.LoadStrategyCode(%s): %v", shortname, e)
	}
	return sc
}
