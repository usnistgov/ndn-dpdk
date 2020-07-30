package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
)

var (
	ifname  = flag.String("i", "", "network interface name (required)")
	rxq     = flag.Int("rxq", 0, "RX queue size")
	txq     = flag.Int("txq", 0, "TX queue size")
	local   macaddr.Flag
	remote  macaddr.Flag
	verbose = flag.Bool("v", false, "print received packet names")
	respond = flag.Bool("respond", false, "respond to every Interest with Data")
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

	for packet := range face.Rx() {
		if *verbose {
			fmt.Println(packet)
		}
		if *respond && packet.Interest != nil {
			face.Tx() <- ndn.MakeData(packet.Interest)
		}
	}
}
