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

	makeRetrieveByFaceID := func(fromGen func(gen *TrafficGen) any) func(id iface.ID) any {
		return func(id iface.ID) any {
			gen := Get(id)
			if gen == nil {
				return nil
			}
			return fromGen(gen)
		}
	}

	tgproducer.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) any { return gen.Producer() })
	fileserver.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) any { return gen.FileServer() })
	tgconsumer.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) any { return gen.Consumer() })
	fetch.GqlRetrieveByFaceID = makeRetrieveByFaceID(func(gen *TrafficGen) any { return gen.Fetcher() })
}
