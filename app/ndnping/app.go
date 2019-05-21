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
	"ndn-dpdk/iface/createface"
)

// LCoreAlloc roles.
const (
	LCoreRole_Input    = iface.LCoreRole_RxLoop
	LCoreRole_Output   = iface.LCoreRole_TxLoop
	LCoreRole_Server   = "SVR"
	LCoreRole_ClientRx = "CLIR"
	LCoreRole_ClientTx = "CLIT"
)

type App struct {
	Tasks []Task
	rxls  []*iface.RxLoop
}

func New(cfg []TaskConfig) (app *App, e error) {
	app = new(App)

	appinit.ProvideCreateFaceMempools()
	for _, numaSocket := range createface.ListRxTxNumaSockets() {
		// TODO create rxl and txl for configured faces only
		rxLCore := dpdk.LCoreAlloc.Alloc(LCoreRole_Input, numaSocket)
		rxl := iface.NewRxLoop(rxLCore.GetNumaSocket())
		rxl.SetLCore(rxLCore)
		app.rxls = append(app.rxls, rxl)
		createface.AddRxLoop(rxl)

		txLCore := dpdk.LCoreAlloc.Alloc(LCoreRole_Output, numaSocket)
		txl := iface.NewTxLoop(txLCore.GetNumaSocket())
		txl.SetLCore(txLCore)
		txl.Launch()
		createface.AddTxLoop(txl)
	}

	for i, taskCfg := range cfg {
		face, e := createface.Create(taskCfg.Face.Locator)
		if e != nil {
			return nil, fmt.Errorf("[%d] face creation error: %v", i, e)
		}
		task, e := newTask(face, taskCfg)
		if e != nil {
			return nil, fmt.Errorf("[%d] init error: %v", i, e)
		}
		app.Tasks = append(app.Tasks, task)
	}

	return app, nil
}

func (app *App) Launch() {
	for _, rxl := range app.rxls {
		app.launchRxl(rxl)
	}
	for _, task := range app.Tasks {
		task.Launch()
	}
}

func (app *App) launchRxl(rxl *iface.RxLoop) {
	hasFace := false
	minFaceId := iface.FACEID_MAX
	maxFaceId := iface.FACEID_MIN
	for _, faceId := range rxl.ListFaces() {
		hasFace = true
		if faceId < minFaceId {
			minFaceId = faceId
		}
		if faceId > maxFaceId {
			maxFaceId = faceId
		}
	}
	if !hasFace {
		return
	}

	inputC := C.PingInput_New(C.uint16_t(minFaceId), C.uint16_t(maxFaceId), C.unsigned(rxl.GetNumaSocket()))
	for i, task := range app.Tasks {
		entryC := C.PingInput_GetEntry(inputC, C.uint16_t(task.Face.GetFaceId()))
		if entryC == nil {
			continue
		}
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

	rxl.SetCallback(unsafe.Pointer(C.PingInput_FaceRx), unsafe.Pointer(inputC))
	rxl.Launch()
}

type Task struct {
	Face   iface.IFace
	Client *Client
	Server *Server
}

func newTask(face iface.IFace, cfg TaskConfig) (task Task, e error) {
	numaSocket := face.GetNumaSocket()
	task.Face = face
	if cfg.Client != nil {
		if task.Client, e = newClient(task.Face, *cfg.Client); e != nil {
			return Task{}, e
		}
		task.Client.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, numaSocket))
		task.Client.Tx.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientTx, numaSocket))
	}
	if cfg.Server != nil {
		if task.Server, e = newServer(task.Face, *cfg.Server); e != nil {
			return Task{}, e
		}
		task.Server.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_Server, numaSocket))
	}
	return task, nil
}

func (task *Task) Launch() {
	if task.Server != nil {
		task.Server.Launch()
	}
	if task.Client != nil {
		task.Client.Launch()
		task.Client.Tx.Launch()
	}
}

func (task *Task) Close() error {
	if task.Server != nil {
		task.Server.Close()
	}
	if task.Client != nil {
		task.Client.Close()
	}
	task.Face.Close()
	return nil
}
