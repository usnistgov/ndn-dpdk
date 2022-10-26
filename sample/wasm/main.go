//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"log"
	"syscall/js"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/wasmtransport"
)

func demo(router, prefix string) {
	tr, e := wasmtransport.NewWebSocket(router)
	if e != nil {
		log.Fatalln(e)
	}

	face, e := l3.NewFace(tr, l3.FaceConfig{})
	if e != nil {
		log.Fatalln(e)
	}

	fwFace, e := l3.GetDefaultForwarder().AddFace(face)
	if e != nil {
		log.Fatalln(e)
	}
	fwFace.AddRoute(ndn.ParseName("/"))

	for t := range time.Tick(time.Second) {
		go func(n int64) {
			interest := ndn.MakeInterest(fmt.Sprintf("%s/%d", prefix, n))
			log.Println("<I", interest)
			t0 := time.Now()
			data, e := endpoint.Consume(context.Background(), interest, endpoint.ConsumerOptions{})
			if e != nil {
				log.Println(">E", e)
			} else {
				log.Printf(">D %v rtt=%dms", data, time.Since(t0).Milliseconds())
			}
		}(t.UnixNano())
	}
}

func main() {
	jsDocument := js.Global().Get("document")
	router := jsDocument.Call("querySelector", "#app_router").Get("value").String()
	prefix := jsDocument.Call("querySelector", "#app_prefix").Get("value").String()
	demo(router, prefix)
}
