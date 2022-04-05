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

	tgproducer.GqlRetrieveByFaceID = makeRetrieveByFaceID(TrafficGen.Producer)
	fileserver.GqlRetrieveByFaceID = makeRetrieveByFaceID(TrafficGen.FileServer)
	tgconsumer.GqlRetrieveByFaceID = makeRetrieveByFaceID(TrafficGen.Consumer)
	fetch.GqlRetrieveByFaceID = makeRetrieveByFaceID(TrafficGen.Fetcher)
}

func makeRetrieveByFaceID[T any](f func(TrafficGen) *T) func(id iface.ID) *T {
	return func(id iface.ID) *T {
		gen := Get(id)
		if gen == nil {
			return nil
		}
		return f(*gen)
	}
}
