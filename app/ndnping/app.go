package ndnping

/*
#include "input.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
)

type App struct {
	Tasks []Task
	rxl   *ethface.RxLoop
	txl   *iface.MultiTxLoop
}

func NewApp(cfg []TaskConfig) (app *App, e error) {
	app = new(App)

	for i, taskCfg := range cfg {
		task, e := newTask(taskCfg)
		if e != nil {
			return nil, fmt.Errorf("[%d] init error: %v", i, e)
		}
		if faceKind := task.Face.GetFaceId().GetKind(); faceKind != iface.FaceKind_Eth {
			return nil, fmt.Errorf("[%d] FaceKind %v is not supported", i, faceKind)
		}
		app.Tasks = append(app.Tasks, task)
	}

	return app, nil
}

func (app *App) getNumaSocket() dpdk.NumaSocket {
	return app.Tasks[0].Face.GetNumaSocket()
}

func (app *App) Launch() {
	app.launchRx()
	app.launchTx()
	for _, task := range app.Tasks {
		task.Launch()
	}
}

func (app *App) launchRx() {
	rxl := ethface.NewRxLoop(len(app.Tasks), app.getNumaSocket())
	minFaceId := iface.FaceId(0xFFFF)
	maxFaceId := iface.FaceId(0x0000)
	for _, task := range app.Tasks {
		rxl.Add(task.Face.(*ethface.EthFace))

		faceId := task.Face.GetFaceId()
		if faceId < minFaceId {
			minFaceId = faceId
		}
		if faceId > maxFaceId {
			maxFaceId = faceId
		}
	}

	inputC := C.NdnpingInput_New(C.uint16_t(minFaceId), C.uint16_t(maxFaceId), C.unsigned(app.getNumaSocket()))
	for i, task := range app.Tasks {
		entryC := C.__NdnpingInput_GetEntry(inputC, C.uint16_t(task.Face.GetFaceId()))
		if task.Client != nil {
			queue, e := dpdk.NewRing(fmt.Sprintf("client-rx-%d", i), 256,
				task.Face.GetNumaSocket(), true, true)
			if e != nil {
				panic(e)
			}
			entryC.clientQueue = (*C.struct_rte_ring)(queue.GetPtr())
			task.Client.c.rxQueue = entryC.clientQueue
		}
		if task.Server != nil {
			queue, e := dpdk.NewRing(fmt.Sprintf("server-rx-%d", i), 256,
				task.Face.GetNumaSocket(), true, true)
			if e != nil {
				panic(e)
			}
			entryC.serverQueue = (*C.struct_rte_ring)(queue.GetPtr())
			task.Server.c.rxQueue = entryC.serverQueue
		}
	}

	appinit.MustLaunch(func() int {
		rxl.RxLoop(64, unsafe.Pointer(C.NdnpingInput_FaceRx), unsafe.Pointer(inputC))
		return 0
	}, app.getNumaSocket())
}

func (app *App) launchTx() {
	txl := iface.NewMultiTxLoop()
	for _, task := range app.Tasks {
		txl.AddFace(task.Face)
	}

	appinit.MustLaunch(func() int {
		txl.TxLoop()
		return 0
	}, app.getNumaSocket())
}

type Task struct {
	Face   iface.IFace
	Client *Client
	Server *Server
}

func newTask(cfg TaskConfig) (task Task, e error) {
	remoteUri, e := faceuri.Parse(cfg.Face.Remote)
	if e != nil {
		return Task{}, fmt.Errorf("faceuri.Parse(remote): %v", e)
	}
	var localUri *faceuri.FaceUri
	if cfg.Face.Local != "" {
		localUri, e = faceuri.Parse(cfg.Face.Local)
		if e != nil {
			return Task{}, fmt.Errorf("faceuri.Parse(local): %v", e)
		}
	}
	task.Face, e = appinit.NewFaceFromUri(remoteUri, localUri)
	if e != nil {
		return Task{}, fmt.Errorf("appinit.NewFaceFromUri: %v", e)
	}
	task.Face.EnableThreadSafeTx(256)

	if cfg.Client != nil {
		task.Client, e = newClient2(task.Face, *cfg.Client)
		if e != nil {
			task.Close()
			return Task{}, e
		}
	}

	if cfg.Server != nil {
		task.Server, e = newServer2(task.Face, *cfg.Server)
		if e != nil {
			task.Close()
			return Task{}, e
		}
	}

	return task, nil
}

func (task *Task) Launch() {
	numaSocket := task.Face.GetNumaSocket()
	if task.Server != nil {
		appinit.MustLaunch(task.Server.Run, numaSocket)
	}
	if task.Client != nil {
		appinit.MustLaunch(task.Client.RunRx, numaSocket)
		appinit.MustLaunch(task.Client.RunTx, numaSocket)
	}
}

func (task *Task) Close() error {
	if task.Server != nil {
		task.Server.Close()
	}
	if task.Client != nil {
		task.Server.Close()
	}
	task.Face.Close()
	return nil
}
