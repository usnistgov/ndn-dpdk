package ping

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

// LCoreAlloc roles.
const (
	LCoreRole_Input    = "RX"
	LCoreRole_Output   = "TX"
	LCoreRole_Server   = "SVR"
	LCoreRole_ClientRx = "CLIR"
	LCoreRole_ClientTx = "CLIT"
)

type App struct {
	Tasks  []Task
	inputs []*Input
}

type Input struct {
	rxl  iface.RxLoop
	face iface.Face
}

func New(cfg []TaskConfig) (app *App, e error) {
	app = new(App)

	iface.ChooseRxLoop = func(rxg iface.RxGroup) iface.RxLoop {
		rxl := iface.NewRxLoop(rxg.NumaSocket())
		ealthread.AllocThread(rxl)

		var input Input
		input.rxl = rxl
		app.inputs = append(app.inputs, &input)
		return rxl
	}

	iface.ChooseTxLoop = func(face iface.Face) iface.TxLoop {
		txl := iface.NewTxLoop(face.NumaSocket())
		ealthread.Launch(txl)
		return txl
	}

	for i, taskCfg := range cfg {
		face, e := createface.Create(taskCfg.Face.Locator)
		if e != nil {
			return nil, fmt.Errorf("[%d] face creation error: %w", i, e)
		}
		if nInputs := len(app.inputs); nInputs == 0 || app.inputs[nInputs-1].face != nil {
			return nil, fmt.Errorf("[%d] unexpected RxLoop creation")
		}
		app.inputs[len(app.inputs)-1].face = face

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
	demuxI := input.rxl.InterestDemux()
	demuxD := input.rxl.DataDemux()
	demuxN := input.rxl.NackDemux()
	demuxI.InitDrop()
	demuxD.InitDrop()
	demuxN.InitDrop()

	for _, task := range app.Tasks {
		if task.Face.ID() != input.face.ID() {
			continue
		}
		task.configureDemux(demuxI, demuxD, demuxN)
	}

	input.rxl.Launch()
}
