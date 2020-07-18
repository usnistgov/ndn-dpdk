package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/afpackettransport"
)

var (
	ifname  = flag.String("i", "", "network interface name")
	rxq     = flag.Int("rxq", 0, "RX queue size")
	txq     = flag.Int("txq", 0, "TX queue size")
	local   macaddr.Flag
	remote  macaddr.Flag
	verbose = flag.Bool("v", false, "print received packet names")
	respond = flag.Bool("respond", false, "respond every Interest with Data")
)

func init() {
	flag.Var(&local, "local", "local MAC address")
	flag.Var(&remote, "remote", "remote MAC address")
}

func main() {
	flag.Parse()
	var cfg afpackettransport.Config
	cfg.Local = local.HardwareAddr
	cfg.Remote = remote.HardwareAddr
	cfg.RxQueueSize = *rxq
	cfg.TxQueueSize = *txq

	tr, e := afpackettransport.New(*ifname, cfg)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	face, e := ndn.NewL3Face(tr)
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
