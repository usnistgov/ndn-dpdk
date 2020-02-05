package ping

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
	Tasks   []Task
	rxls    []*iface.RxLoop
	initCfg InitConfig
}

func New(cfg []TaskConfig, initCfg InitConfig) (app *App, e error) {
	app = new(App)
	app.initCfg = initCfg

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
	for _, task := range app.Tasks {
		task.connectInput(inputC)
	}

	rxl.SetCallback(unsafe.Pointer(C.PingInput_FaceRx), unsafe.Pointer(inputC))
	rxl.Launch()
}

func (app *App) makeRxQueue(id string, numaSocket dpdk.NumaSocket) (queue dpdk.Ring) {
	queue, e := dpdk.NewRing(id, app.initCfg.QueueCapacity, numaSocket, true, true)
	if e != nil {
		panic(e)
	}
	return queue
}
