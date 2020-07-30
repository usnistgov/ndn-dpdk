package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
)

var (
	ifname     = flag.String("i", "", "network interface name (required)")
	rxq        = flag.Int("rxq", l3.DefaultTransportRxQueueSize, "RX queue size")
	txq        = flag.Int("txq", l3.DefaultTransportTxQueueSize, "TX queue size")
	local      macaddr.Flag
	remote     macaddr.Flag
	dump       = flag.Bool("dump", false, "print received packet names")
	respond    = flag.Bool("respond", false, "respond to every Interest with Data")
	payloadlen = flag.Int("payloadlen", 0, "Data payload length for -respond")
	transmit   = flag.Duration("transmit", 0, "transmit Interests at given interval")
	prefix     = flag.String("prefix", fmt.Sprintf("/ndndpdk/%d", time.Now().Unix()), "Interest name prefix for -transmit")
)

func init() {
	flag.Var(&local, "local", "local MAC address")
	flag.Var(&remote, "remote", "remote MAC address")
}

func main() {
	flag.Parse()
	if *ifname == "" {
		flag.Usage()
		os.Exit(2)
	}

	var cfg afpacket.Config
	cfg.Local = local.HardwareAddr
	cfg.Remote = remote.HardwareAddr
	cfg.RxQueueSize = *rxq
	cfg.TxQueueSize = *txq

	tr, e := afpacket.New(*ifname, cfg)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	face, e := l3.NewFace(tr)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	go func() {
		payload := make([]byte, *payloadlen)
		for packet := range face.Rx() {
			if *dump {
				fmt.Println(packet)
			}
			if *respond && packet.Interest != nil {
				face.Tx() <- ndn.MakeData(packet.Interest, payload)
			}
		}
	}()

	go func() {
		if *transmit <= 0 {
			return
		}
		for t := range time.Tick(*transmit) {
			name := fmt.Sprintf("%s/%d", *prefix, t.UnixNano())
			face.Tx() <- ndn.MakeInterest(name, ndn.MustBeFreshFlag)
		}
	}()

	select {}
}
