// Package tg is a traffic generator.
package tg

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// LCoreAlloc roles.
const (
	roleInput    = "RX"
	roleOutput   = "TX"
	roleProducer = "PRODUCER"
	roleConsumer = "CONSUMER"
)

// App represents the traffic generator.
type App struct {
	Tasks  []*Task
	inputs []*Input
}

// Input represents an input thread.
type Input struct {
	rxl  iface.RxLoop
	face iface.Face
}

// New creates an App.
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
		face, e := taskCfg.Face.Locator.CreateFace()
		if e != nil {
			return nil, fmt.Errorf("[%d] face creation error: %w", i, e)
		}
		if nInputs := len(app.inputs); nInputs == 0 || app.inputs[nInputs-1].face != nil {
			return nil, fmt.Errorf("[%d] unexpected RxLoop creation", i)
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

// Launch starts the traffic generator.
func (app *App) Launch() {
	for _, input := range app.inputs {
		app.launchInput(input)
	}
	for _, task := range app.Tasks {
		task.launch()
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

// Stop stops and closes the traffic generator.
func (app *App) Close() error {
	for _, task := range app.Tasks {
		task.close()
	}
	for _, input := range app.inputs {
		input.rxl.Stop()
		input.rxl.Close()
	}
	return nil
}
