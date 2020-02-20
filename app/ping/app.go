package ping

import (
	"fmt"

	"ndn-dpdk/app/inputdemux"
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
	Tasks  []Task
	inputs []*Input
}

type Input struct {
	rxl    *iface.RxLoop
	demux3 inputdemux.Demux3
}

func New(cfg []TaskConfig) (app *App, e error) {
	app = new(App)

	appinit.ProvideCreateFaceMempools()

	createface.CustomGetRxl = func(rxg iface.IRxGroup) *iface.RxLoop {
		lc := dpdk.LCoreAlloc.Alloc(LCoreRole_Input, rxg.GetNumaSocket())
		socket := lc.GetNumaSocket()
		rxl := iface.NewRxLoop(socket)
		rxl.SetLCore(lc)

		var input Input
		input.rxl = rxl
		app.inputs = append(app.inputs, &input)

		createface.AddRxLoop(rxl)
		return rxl
	}

	createface.CustomGetTxl = func(face iface.IFace) *iface.TxLoop {
		lc := dpdk.LCoreAlloc.Alloc(LCoreRole_Output, face.GetNumaSocket())
		socket := lc.GetNumaSocket()
		txl := iface.NewTxLoop(socket)
		txl.SetLCore(lc)
		txl.Launch()

		createface.AddTxLoop(txl)
		return txl
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
	for _, input := range app.inputs {
		app.launchInput(input)
	}
	for _, task := range app.Tasks {
		task.Launch()
	}
}

func (app *App) launchInput(input *Input) {
	faces := input.rxl.ListFaces()
	if len(faces) != 1 {
		panic("RxLoop should have exactly one face")
	}

	input.demux3 = inputdemux.NewDemux3(input.rxl.GetNumaSocket())
	demuxI := input.demux3.GetInterestDemux()
	demuxI.InitDrop()
	demuxD := input.demux3.GetDataDemux()
	demuxD.InitDrop()
	demuxN := input.demux3.GetNackDemux()
	demuxN.InitDrop()

	for _, task := range app.Tasks {
		if task.Face.GetFaceId() != faces[0] {
			continue
		}

		if task.Server != nil {
			demuxI.InitFirst()
			demuxI.SetDest(0, task.Server.GetRxQueue())
		}

		if task.Client != nil {
			demuxD.InitFirst()
			demuxD.SetDest(0, task.Client.GetRxQueue())
			demuxN.InitFirst()
			demuxN.SetDest(0, task.Client.GetRxQueue())
		} else if task.Fetch != nil {
			demuxD.InitFirst()
			demuxD.SetDest(0, task.Fetch.GetRxQueue())
			demuxN.InitFirst()
			demuxN.SetDest(0, task.Fetch.GetRxQueue())
		}
	}

	input.rxl.SetCallback(inputdemux.Demux3_FaceRx, input.demux3.GetPtr())
	input.rxl.Launch()
}
