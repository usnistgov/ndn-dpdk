package main

import (
	"math/rand"
	"time"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/socketface"
)

var (
	theSocketFaceNumaSocket dpdk.NumaSocket
	theSocketRxg            *socketface.RxGroup
	theSocketTxl            *iface.MultiTxLoop
	theDp                   *fwdp.DataPlane
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	appinit.InitEal()
	initCfg, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}
	initCfg.Mempool.Apply()
	initCfg.FaceQueueCapacity.Apply()

	startDp(initCfg.Ndt, initCfg.Fib, initCfg.Fwdp)
	startMgmt()

	select {}
}

func startDp(ndtCfg ndt.Config, fibCfg fib.Config, dpInit fwdpInitConfig) {
	log.WithField("nSlaves", len(appinit.Eal.Slaves)).Info("EAL ready")
	lcr := appinit.NewLCoreReservations()

	var dpCfg fwdp.Config
	var outputLCores []dpdk.LCore
	var outputTxLoopers []iface.ITxLooper

	dpCfg.Ndt = ndtCfg
	dpCfg.Fib = fibCfg

	// create EthFaces
	{
		nRxThreads := dpInit.EthInputsPerFace
		if nRxThreads == 0 {
			nRxThreads = dpInit.EthInputsPerNuma
		}
		rxlPerNuma := make(map[dpdk.NumaSocket][]*ethface.RxLoop)
		txlPerNuma := make(map[dpdk.NumaSocket]*iface.MultiTxLoop)

		ethDevs := dpdk.ListEthDevs()
		for _, port := range ethDevs {
			logEntry := log.WithFields(makeLogFields("port", port, "name", port.GetName()))
			face, e := appinit.NewFaceFromEthDev(port, nRxThreads)
			if e != nil {
				logEntry.WithError(e).Fatal("EthFace creation error")
				continue
			}

			socket := face.GetNumaSocket()
			if socket == dpdk.NUMA_SOCKET_ANY {
				socket = 0
			}
			logEntry = logEntry.WithField("socket", socket)

			if dpInit.EthInputsPerFace > 0 {
				lcores := make([]dpdk.LCore, dpInit.EthInputsPerFace)
				for i := range lcores {
					lcores[i] = lcr.MustReserve(face.GetNumaSocket())
					dpCfg.InputLCores = append(dpCfg.InputLCores, lcores[i])
					dpCfg.InputRxLoopers = append(dpCfg.InputRxLoopers, appinit.MakeRxLooper(face))
				}
				logEntry = logEntry.WithField("rx-lcores", lcores)
			} else {
				rxls, ok := rxlPerNuma[socket]
				if !ok {
					lcores := make([]dpdk.LCore, dpInit.EthInputsPerNuma)
					rxls = make([]*ethface.RxLoop, dpInit.EthInputsPerNuma)
					for i := range rxls {
						lcores[i] = lcr.MustReserve(socket)
						rxls[i] = ethface.NewRxLoop(len(ethDevs), socket)
						dpCfg.InputLCores = append(dpCfg.InputLCores, lcores[i])
						dpCfg.InputRxLoopers = append(dpCfg.InputRxLoopers, rxls[i])
					}
					rxlPerNuma[socket] = rxls
					logEntry = logEntry.WithField("shared-rx-lcores", lcores)
				} else {
					logEntry = logEntry.WithField("shared-rx-lcores", "reuse")
				}

				for _, rxl := range rxls {
					if e := rxl.Add(face.(*ethface.EthFace)); e != nil {
						logEntry.WithError(e).Fatal("rxl.Add failed")
					}
				}
			}

			if e := face.EnableThreadSafeTx(appinit.TheFaceQueueCapacityConfig.EthTxPkts); e != nil {
				logEntry.WithError(e).Fatal("EnableThreadSafeTx failed")
			}

			if !dpInit.EthShareTx {
				lcore := lcr.MustReserve(socket)
				logEntry = logEntry.WithField("tx-lcore", lcore)
				outputLCores = append(outputLCores, lcore)
				outputTxLoopers = append(outputTxLoopers, appinit.MakeTxLooper(face))
			} else {
				txl, ok := txlPerNuma[socket]
				if !ok {
					lcore := lcr.MustReserve(socket)
					txl = iface.NewMultiTxLoop()
					txlPerNuma[socket] = txl
					outputLCores = append(outputLCores, lcore)
					outputTxLoopers = append(outputTxLoopers, txl)
					logEntry = logEntry.WithField("shared-tx-lcore", lcore)
				} else {
					logEntry = logEntry.WithField("shared-tx-lcore", "reuse")
				}
				txl.AddFace(face)
			}

			logEntry.Info("EthFace created")
		}
	}

	// prepare SocketFaces
	if dpInit.EnableSocketFace {
		theSocketRxg = socketface.NewRxGroup()
		inputLc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
		theSocketFaceNumaSocket = inputLc.GetNumaSocket()
		dpCfg.InputLCores = append(dpCfg.InputLCores, inputLc)
		dpCfg.InputRxLoopers = append(dpCfg.InputRxLoopers, theSocketRxg)

		theSocketTxl = iface.NewMultiTxLoop()
		outputLc := lcr.MustReserve(theSocketFaceNumaSocket)
		outputLCores = append(outputLCores, outputLc)
		outputTxLoopers = append(outputTxLoopers, theSocketTxl)

		log.WithFields(makeLogFields("rx-lcore", inputLc, "socket", theSocketFaceNumaSocket,
			"tx-lcore", outputLc)).Info("SocketFaces ready")
	}

	// enable crypto thread
	{
		lc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
		dpCfg.CryptoLCore = lc
		dpCfg.Crypto.InputCapacity = 64
		dpCfg.Crypto.OpPoolCapacity = 1023
		dpCfg.Crypto.OpPoolCacheSize = 31
		log.WithFields(makeLogFields("lcore", lc, "socket", lc.GetNumaSocket())).Info("crypto-helper created")
	}

	// allocate forwarding threads
	nFwds := len(appinit.Eal.Slaves) - len(dpCfg.InputLCores) - len(outputLCores) - 1
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

	// set dataplane config
	dpCfg.FwdQueueCapacity = dpInit.FwdQueueCapacity
	dpCfg.LatencySampleFreq = dpInit.LatencySampleFreq
	dpCfg.Pcct.MaxEntries = dpInit.PcctCapacity
	dpCfg.Pcct.CsCapacity = dpInit.CsCapacity

	// create dataplane
	{
		var e error
		theDp, e = fwdp.New(dpCfg)
		if e != nil {
			log.WithError(e).Fatal("dataplane init error")
		}
	}

	// launch output threads
	for i := range outputTxLoopers {
		txl := outputTxLoopers[i]
		outputLCores[i].RemoteLaunch(func() int {
			txl.TxLoop()
			return 0
		})
	}

	// launch dataplane
	if e := theDp.Launch(); e != nil {
		log.WithError(e).Fatal("dataplane launch error")
	}
	log.Info("dataplane started")
}
