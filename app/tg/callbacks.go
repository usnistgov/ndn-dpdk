package tg

import (
	"reflect"

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

	makeRetrieveByFaceID := func(methodName string) func(id iface.ID) interface{} {
		typ := reflect.TypeOf(&TrafficGen{})
		method, _ := typ.MethodByName(methodName)
		return func(id iface.ID) interface{} {
			gen := Get(id)
			if gen == nil {
				return nil
			}
			return method.Func.Call([]reflect.Value{reflect.ValueOf(gen)})
		}
	}

	tgproducer.GqlRetrieveByFaceID = makeRetrieveByFaceID("Producer")
	fileserver.GqlRetrieveByFaceID = makeRetrieveByFaceID("FileServer")
	tgconsumer.GqlRetrieveByFaceID = makeRetrieveByFaceID("Consumer")
	fetch.GqlRetrieveByFaceID = makeRetrieveByFaceID("Fetcher")
}
