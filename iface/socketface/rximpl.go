package socketface

import (
	"sync"
	"sync/atomic"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/zyedidia/generic/mapset"
	"go.uber.org/zap"
)

type rxGroup interface {
	iface.RxGroup
	close()
	run(face *socketFace) error
}

type rxImpl struct {
	describe string
	nilValue any
	instance atomic.Value
	nFaces   atomic.Int32
	create   func() (rxGroup, error)
}

func (impl *rxImpl) String() string {
	return impl.describe
}

func (impl *rxImpl) start(face *socketFace) error {
	id, ctx := face.ID(), face.transport.Context()

	if impl.nFaces.Add(1) == 1 {
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
	if impl.nFaces.Add(-1) > 0 {
		return
	}
	rxg := impl.instance.Swap(impl.nilValue).(rxGroup)
	rxg.close()
}

type rxFaceList struct {
	set  *mapset.Set[*socketFace]
	lock sync.RWMutex
}

func (fl *rxFaceList) faceListPut(face *socketFace) func() {
	fl.lock.Lock()
	defer fl.lock.Unlock()
	if fl.set == nil {
		s := mapset.New[*socketFace]()
		fl.set = &s
	}
	fl.set.Put(face)
	return func() {
		fl.lock.Lock()
		defer fl.lock.Unlock()
		fl.set.Remove(face)
	}
}

// Faces implements RxGroup interface.
func (fl *rxFaceList) Faces() (list []iface.Face) {
	fl.lock.RLock()
	defer fl.lock.RUnlock()
	fl.set.Each(func(face *socketFace) {
		list = append(list, face)
	})
	return list
}
