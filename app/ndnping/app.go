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
	LCoreRole_Server   = "SVR"
	LCoreRole_ClientRx = "CLIR"
	LCoreRole_ClientTx = "CLIT"
)

type App struct {
	Tasks []Task
	rxls  []*iface.RxLoop
}

func NewApp(cfg []TaskConfig) (app *App, e error) {
	app = new(App)

	appinit.BeforeStartRxl = app.addRxl
	appinit.WantLaunchRxl = false

	var faceLocators []iface.Locator
	for _, taskCfg := range cfg {
		faceLocators = append(faceLocators, taskCfg.Face.Locator)
	}
	faces, e := createface.Create(faceLocators...)
	if e != nil {
		return nil, e
	}

	for i, taskCfg := range cfg {
		task, e := newTask(taskCfg, faces[i])
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

func (app *App) addRxl(rxl *iface.RxLoop) (usr interface{}, e error) {
	app.rxls = append(app.rxls, rxl)
	return nil, nil
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
	minFaceId := iface.FACEID_MAX
	maxFaceId := iface.FACEID_MIN
	for _, faceId := range rxl.ListFaces() {
		if faceId < minFaceId {
			minFaceId = faceId
		}
		if faceId > maxFaceId {
			maxFaceId = faceId
		}
	}

	inputC := C.NdnpingInput_New(C.uint16_t(minFaceId), C.uint16_t(maxFaceId), C.unsigned(rxl.GetNumaSocket()))
	for i, task := range app.Tasks {
		entryC := C.__NdnpingInput_GetEntry(inputC, C.uint16_t(task.Face.GetFaceId()))
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

	rxl.SetCallback(unsafe.Pointer(C.NdnpingInput_FaceRx), unsafe.Pointer(inputC))
	rxl.Launch()
}

type Task struct {
	Face   iface.IFace
	Client *Client
	Server *Server
}

func newTask(cfg TaskConfig, face iface.IFace) (task Task, e error) {
	numaSocket := face.GetNumaSocket()
	task.Face = face
	if cfg.Client != nil {
		task.Client = newClient(task.Face, *cfg.Client)
		task.Client.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, numaSocket))
		task.Client.Tx.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientTx, numaSocket))
	}
	if cfg.Server != nil {
		task.Server = newServer(task.Face, *cfg.Server)
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
