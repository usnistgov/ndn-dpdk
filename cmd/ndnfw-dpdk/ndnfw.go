package main

import (
	"math/rand"
	"time"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/faceuri"
)

var theDp *fwdp.DataPlane

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	appinit.InitEal()
	initCfg, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}
	initCfg.Mempool.Apply()

	startDp(initCfg.Ndt, initCfg.Fib, initCfg.Fwdp)
	startFaces(initCfg.Face, initCfg.Fwdp.AutoFaces)
	startMgmt()

	select {}
}

func startDp(ndtCfg ndt.Config, fibCfg fib.Config, dpInit fwdpInitConfig) {
	log.WithField("nSlaves", len(appinit.Eal.Slaves)).Info("EAL ready")
	lcr := appinit.NewLCoreReservations()
	appinit.TxlLCoreReservation = lcr

	var dpCfg fwdp.Config
	dpCfg.Ndt = ndtCfg
	dpCfg.Fib = fibCfg

	// assign input lcores
	if len(dpInit.InputLCores) == 0 {
		log.Fatal("no lcore reserved for input")
	}
	dpCfg.InputLCores = dpInit.InputLCores
	lcr.MarkReserved(dpCfg.InputLCores...)

	// enable crypto thread
	dpCfg.CryptoLCore = dpdk.LCORE_INVALID
	if len(dpInit.CryptoLCores) > 0 {
		dpCfg.CryptoLCore = dpInit.CryptoLCores[0]
		dpCfg.Crypto.InputCapacity = 64
		dpCfg.Crypto.OpPoolCapacity = 1023
		dpCfg.Crypto.OpPoolCacheSize = 31
	}
	lcr.MarkReserved(dpCfg.CryptoLCore)

	// assign forwarding lcores
	if len(dpInit.FwdLCores) == 0 {
		log.Fatal("no lcore reserved for forwarding")
	}
	dpCfg.FwdLCores = dpInit.FwdLCores
	lcr.MarkReserved(dpCfg.FwdLCores...)

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

	// launch dataplane
	if e := theDp.Launch(); e != nil {
		log.WithError(e).Fatal("dataplane launch error")
	}
	log.Info("dataplane started")
}

func startFaces(faceCfg createface.Config, wantAutoFaces bool) {
	if e := appinit.EnableCreateFace(faceCfg); e != nil {
		log.WithError(e).Fatal("face init error")
	}

	if wantAutoFaces {
		for _, ethdev := range dpdk.ListEthDevs() {
			var a createface.CreateArg
			a.Remote = faceuri.MustMakeEtherUri(ethdev.GetName(), nil, 0)
			a.Local = faceuri.MustMakeEtherUri(ethdev.GetName(), ethdev.GetMacAddr(), 0)
			if _, e := createface.Create(a); e != nil {
				log.WithError(e).WithField("ethdev", ethdev).Fatal("auto-face create error")
			}
		}
	}
}
