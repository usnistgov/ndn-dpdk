package main

import (
	// "fmt"
	// stdlog "log"
	// "os"
	// "time"

	"ndn-dpdk/app/dump"
	// "ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const Dump_RingCapacity = 256

type PktcopyProc struct {
	face      iface.IFace
	pcrx      *PktcopyRx
	rxLcore   dpdk.LCore
	txLcore   dpdk.LCore
	dumper    *dump.Dump
	dumpLcore dpdk.LCore
}

func main() {
	// appinit.InitEal()
	// pc, e := ParseCommand(appinit.Eal.Args[1:])
	// if e != nil {
	// 	log.WithError(e).Fatal("command line error")
	// }

	// // initialize faces, PktcopyRxs, and dumpers
	// lcr := appinit.NewLCoreReservations()
	// procs := make([]PktcopyProc, len(pc.Faces))
	// for i, faceUri := range pc.Faces {
	// 	logEntry := log.WithField("face", faceUri)
	// 	proc := &procs[i]
	// 	face, e := appinit.NewFaceFromUri(faceUri, nil)
	// 	if e != nil {
	// 		logEntry.WithError(e).Fatal("face creation error")
	// 	}
	// 	face.EnableThreadSafeTx(appinit.TheFaceQueueCapacityConfig.EthTxPkts)
	// 	numaSocket := face.GetNumaSocket()
	// 	proc.face = face

	// 	pcrx := NewPktcopyRx(face)
	// 	proc.pcrx = pcrx

	// 	if pc.Dump {
	// 		ringName := fmt.Sprintf("dump_%d", i)
	// 		ring, e := dpdk.NewRing(ringName, Dump_RingCapacity, numaSocket, true, true)
	// 		if e != nil {
	// 			logEntry.WithField("ring", ringName).WithError(e).Fatal("dump ring creation error")
	// 		}
	// 		pcrx.SetDumpRing(ring)

	// 		prefix := fmt.Sprintf("%d ", face.GetFaceId())
	// 		logger := stdlog.New(os.Stderr, prefix, stdlog.Lmicroseconds)
	// 		proc.dumper = dump.New(ring, logger)
	// 	}

	// 	proc.rxLcore = lcr.MustReserve(numaSocket)
	// 	proc.txLcore = lcr.MustReserve(numaSocket)
	// }
	// if pc.Dump {
	// 	for i := range procs {
	// 		procs[i].dumpLcore = lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
	// 	}
	// }

	// // link PktcopyRx to TX faces
	// switch pc.Mode {
	// case TopoMode_Pair:
	// 	for i := 0; i < len(procs); i += 2 {
	// 		procs[i].pcrx.AddTxFace(procs[i+1].face)
	// 		procs[i+1].pcrx.AddTxFace(procs[i].face)
	// 	}
	// case TopoMode_All:
	// 	for i := range procs {
	// 		for j := range procs {
	// 			if i == j {
	// 				continue
	// 			}
	// 			procs[i].pcrx.AddTxFace(procs[j].face)
	// 		}
	// 	}
	// case TopoMode_OneWay:
	// 	for i := 1; i < len(procs); i++ {
	// 		procs[0].pcrx.AddTxFace(procs[i].face)
	// 	}
	// }

	// // print counters
	// tick := time.Tick(pc.CntInterval)
	// go func() {
	// 	for {
	// 		<-tick
	// 		for it := iface.IterFaces(); it.Valid(); it.Next() {
	// 			stdlog.Printf("%d %v", it.Id, it.Face.ReadCounters())
	// 		}
	// 	}
	// }()

	// // launch
	// for _, proc := range procs {
	// 	txl := appinit.MakeTxLooper(proc.face)
	// 	proc.txLcore.RemoteLaunch(func() int {
	// 		txl.TxLoop()
	// 		return 0
	// 	})
	// 	if proc.dumper != nil {
	// 		proc.dumpLcore.RemoteLaunch(proc.dumper.Run)
	// 	}
	// }
	// for _, proc := range procs {
	// 	proc.rxLcore.RemoteLaunch(proc.pcrx.Run)
	// }

	// select {}
}
