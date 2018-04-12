package dump

import (
	"log"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

const (
	burstSize = 16
)

type Dump struct {
	r    dpdk.Ring
	w    *log.Logger
	stop chan struct{}
}

func New(r dpdk.Ring, w *log.Logger) *Dump {
	var dump Dump
	dump.r = r
	dump.w = w
	dump.stop = make(chan struct{})
	return &dump
}

func (dump *Dump) Close() error {
	dump.stop <- struct{}{}
	return nil
}

func (dump *Dump) Run() int {
	npkts := make([]ndn.Packet, burstSize)
	for {
		count, _ := dump.r.BurstDequeue(npkts)
		for _, npkt := range npkts[:count] {
			dump.w.Print(npkt.String())
			npkt.AsDpdkPacket().Close()
		}

		select {
		case <-dump.stop:
			return 0
		default:
		}
	}
}
