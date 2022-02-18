package tg

import (
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface"
)

func init() {
	iface.OnFaceClosing(func(id iface.ID) {
		if Get(id) != nil {
			logger.Panic("face closing requested while traffic generator is active", id.ZapField("face"))
		}
	})

	makeRetrieveByFaceID := func(fromGen func(gen *TrafficGen) interface{}) func(id iface.ID) interface{} {
		return func(id iface.ID) interface{} {
			gen := Get(id)
			if gen == nil {
				return nil
			}
			return fromGen(gen)
		}
	}

	tgproducer.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) interface{} { return gen.Producer() })
	fileserver.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) interface{} { return gen.FileServer() })
	tgconsumer.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) interface{} { return gen.Consumer() })
	fetch.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) interface{} { return gen.Fetcher() })
}
