// Package tg is a traffic generator.
package tg

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

// TrafficGen represents the traffic generator.
type TrafficGen struct {
	Tasks  []*Task
	inputs []*Input
}

// Input represents an input thread.
type Input struct {
	rxl  iface.RxLoop
	face iface.Face
}

// New creates an App.
func New(cfg []TaskConfig) (gen *TrafficGen, e error) {
	gen = &TrafficGen{}

	iface.ChooseRxLoop = func(rxg iface.RxGroup) iface.RxLoop {
		rxl := iface.NewRxLoop(rxg.NumaSocket())
		ealthread.AllocThread(rxl)

		var input Input
		input.rxl = rxl
		gen.inputs = append(gen.inputs, &input)
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
		if nInputs := len(gen.inputs); nInputs == 0 || gen.inputs[nInputs-1].face != nil {
			return nil, fmt.Errorf("[%d] unexpected RxLoop creation", i)
		}
		gen.inputs[len(gen.inputs)-1].face = face

		task, e := newTask(face, taskCfg)
		if e != nil {
			return nil, fmt.Errorf("[%d] init error: %v", i, e)
		}
		gen.Tasks = append(gen.Tasks, task)
	}

	return gen, nil
}

// Task returns Task on face.
func (gen *TrafficGen) Task(id iface.ID) *Task {
	for _, task := range gen.Tasks {
		if task.Face.ID() == id {
			return task
		}
	}
	return nil
}

// Launch starts the traffic generator.
func (gen *TrafficGen) Launch() {
	for _, input := range gen.inputs {
		gen.launchInput(input)
	}
	for _, task := range gen.Tasks {
		task.launch()
	}
}

func (gen *TrafficGen) launchInput(input *Input) {
	demuxI := input.rxl.InterestDemux()
	demuxD := input.rxl.DataDemux()
	demuxN := input.rxl.NackDemux()
	demuxI.InitDrop()
	demuxD.InitDrop()
	demuxN.InitDrop()

	for _, task := range gen.Tasks {
		if task.Face.ID() != input.face.ID() {
			continue
		}
		task.configureDemux(demuxI, demuxD, demuxN)
	}

	input.rxl.Launch()
}

// Stop stops and closes the traffic generator.
func (gen *TrafficGen) Close() error {
	errs := []error{}
	for _, task := range gen.Tasks {
		errs = append(errs, task.close())
	}
	for _, input := range gen.inputs {
		errs = append(errs,
			input.rxl.Stop(),
			input.rxl.Close(),
		)
	}
	return multierr.Combine(errs...)
}
