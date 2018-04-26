package main

import (
	"errors"
	"math/rand"
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
	"ndn-dpdk/mgmt/ndtmgmt"
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
	log.WithField("nSlaves", len(appinit.Eal.Slaves)).Info("EAL ready")
	lcr := appinit.NewLCoreReservations()

	var dpCfg fwdp.Config

	// reserve lcores for EthFace
	var inputNumaSockets []dpdk.NumaSocket
	var inputRxLoopers []iface.IRxLooper
	var outputLCores []dpdk.LCore
	var outputTxLoopers []iface.ITxLooper
	for _, port := range dpdk.ListEthDevs() {
		logEntry := log.WithField("port", port)
		face, e := appinit.NewFaceFromEthDev(port)
		if e != nil {
			logEntry.WithError(e).Fatal("EthFace creation error")
			continue
		}
		inputLc := lcr.MustReserve(face.GetNumaSocket())
		socket := inputLc.GetNumaSocket()
		logEntry = logEntry.WithFields(makeLogFields("face", face.GetFaceId(), "rx-lcore", inputLc, "socket", socket))
		dpCfg.InputLCores = append(dpCfg.InputLCores, inputLc)
		inputNumaSockets = append(inputNumaSockets, socket)
		inputRxLoopers = append(inputRxLoopers, appinit.MakeRxLooper(face))

		e = face.EnableThreadSafeTx(256)
		if e != nil {
			logEntry.WithError(e).Fatal("EnableThreadSafeTx failed")
		}

		outputLc := lcr.MustReserve(socket)
		logEntry.WithField("tx-lcore", outputLc).Info("EthFace created")
		outputLCores = append(outputLCores, outputLc)
		outputTxLoopers = append(outputTxLoopers, appinit.MakeTxLooper(face))
	}

	// reserve lcore for SocketFaces
	{
		theSocketRxg = socketface.NewRxGroup()
		inputLc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
		socket := inputLc.GetNumaSocket()
		dpCfg.InputLCores = append(dpCfg.InputLCores, inputLc)
		inputNumaSockets = append(inputNumaSockets, socket)
		inputRxLoopers = append(inputRxLoopers, theSocketRxg)

		theSocketTxl = iface.NewMultiTxLoop()
		outputLc := lcr.MustReserve(socket)
		outputLCores = append(outputLCores, outputLc)
		outputTxLoopers = append(outputTxLoopers, theSocketTxl)

		log.WithFields(makeLogFields("rx-lcore", inputLc, "socket", socket, "tx-lcore", outputLc)).Info("SocketFaces ready")
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
			log.WithError(e).Fatal("FIB creation failed")
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
		log.WithFields(makeLogFields("lcore", lc, "socket", lc.GetNumaSocket())).Info("fwd created")
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}
	nFwds = len(dpCfg.FwdLCores)
	if nFwds <= 0 {
		log.Fatal("no lcore available for forwarding")
	}

	// randomize NDT
	theNdt.Randomize(nFwds)

	// set forwarding process config
	dpCfg.FwdQueueCapacity = 128
	dpCfg.LatencySampleRate = 16
	dpCfg.PcctCfg.MaxEntries = 131071
	dpCfg.CsCapacity = 32768

	// create dataplane
	{
		var e error
		theDp, e = fwdp.New(dpCfg)
		if e != nil {
			log.WithError(e).Fatal("dataplane init error")
		}
	}

	// launch output lcores
	log.Info("launching output lcores")
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
	log.Info("launching forwarding lcores")
	for i := range dpCfg.FwdLCores {
		e := theDp.LaunchFwd(i)
		if e != nil {
			log.WithError(e).WithField("i", i).Fatal("fwd launch failed")
		}
	}

	// launch input lcores
	log.Info("launching input lcores")
	const burstSize = 64
	for i, rxl := range inputRxLoopers {
		e := theDp.LaunchInput(i, rxl, burstSize)
		if e != nil {
			log.WithError(e).WithField("i", i).Fatal("input launch failed")
		}
	}

	log.Info("dataplane started")
}

func createFace(remote, local *faceuri.FaceUri) (iface.FaceId, error) {
	if remote.Scheme != "udp4" && remote.Scheme != "tcp4" {
		return iface.FACEID_INVALID, errors.New("face creation only allows udp4 and tcp4 schemes")
	}

	face, e := appinit.NewFaceFromUri(remote, local)
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
	appinit.RegisterMgmt(ndtmgmt.NdtMgmt{theNdt})
	fibmgmt.TheStrategy = theStrategy
	appinit.RegisterMgmt(fibmgmt.FibMgmt{theFib})
	appinit.RegisterMgmt(fwdpmgmt.DpInfoMgmt{theDp})
	appinit.StartMgmt()
}

func loadStrategy(shortname string) fib.StrategyCode {
	logEntry := log.WithField("strategy", shortname)

	elf, e := strategy_elf.Load(shortname)
	if e != nil {
		logEntry.WithError(e).Fatal("strategy ELF load error")
	}
	sc, e := theFib.LoadStrategyCode(elf)
	if e != nil {
		logEntry.WithError(e).Fatal("strategy code load error")
	}

	logEntry.Debug("strategy loaded")
	return sc
}
