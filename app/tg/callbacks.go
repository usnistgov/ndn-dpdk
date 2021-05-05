package tg

import (
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

func init() {
	iface.OnFaceClosing(func(id iface.ID) {
		if Get(id) != nil {
			logger.Panic("face closing requested while traffic generator is active", zap.Uint16("faceID", uint16(id)))
		}
	})

	tgconsumer.GqlRetrieveByFaceID = func(id iface.ID) interface{} {
		gen := Get(id)
		if gen == nil {
			return nil
		}
		return gen.consumer
	}

	tgproducer.GqlRetrieveByFaceID = func(id iface.ID) interface{} {
		gen := Get(id)
		if gen == nil {
			return nil
		}
		return gen.producer
	}

	fetch.GqlRetrieveByFaceID = func(id iface.ID) interface{} {
		gen := Get(id)
		if gen == nil {
			return nil
		}
		return gen.fetcher
	}
}
