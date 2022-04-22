package socketface

import (
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type rxGroup interface {
	iface.RxGroup
	close()
	run(face *socketFace) error
}

type rxImpl struct {
	nilValue any
	instance atomic.Value
	nFaces   atomic.Int32
	create   func() (rxGroup, error)
}

func (impl *rxImpl) start(face *socketFace) error {
	id, ctx := face.ID(), face.transport.Context()

	if impl.nFaces.Inc() == 1 {
		rxg, e := impl.create()
		if e != nil {
			return e
		}
		impl.instance.Store(rxg)
	}

	go func() {
		defer impl.stop()
		if rxg, _ := impl.instance.Load().(rxGroup); rxg != nil {
			if e := rxg.run(face); e != nil && ctx.Err() == nil {
				logger.Error("face RX stopped with error", id.ZapField("id"), zap.Error(e))
			}
		}
	}()

	return nil
}

func (impl *rxImpl) stop() {
	if impl.nFaces.Dec() > 0 {
		return
	}
	rxg := impl.instance.Swap(impl.nilValue).(rxGroup)
	rxg.close()
}
