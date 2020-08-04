package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
)

var (
	gqluri     = flag.String("gqlserver", "http://127.0.0.1:3030/", "GraphQL API of local forwarder")
	ifname     = flag.String("i", "", "network interface name")
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

func createFaceLocal() (face l3.Face, cleanup func()) {
	c, e := gqlmgmt.New(*gqluri)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	f, e := c.OpenFace()
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	log.Println("Opening memif face on local forwarder", f.ID())
	return f.Face(), func() {
		time.Sleep(100 * time.Millisecond) // allow time to close face
		c.Close()
	}
}

func createFaceNetif() (face l3.Face, cleanup func()) {
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

	face, e = l3.NewFace(tr)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}

	log.Println("Opening AF_PACKET face on network interface", *ifname)
	return face, func() {}
}

func main() {
	flag.Parse()
	var face l3.Face
	var cleanup func()
	if *ifname == "" {
		face, cleanup = createFaceLocal()
	} else {
		face, cleanup = createFaceNetif()
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		payload := make([]byte, *payloadlen)
		for packet := range face.Rx() {
			if *dump {
				fmt.Println(packet)
			}
			if *respond && packet.Interest != nil {
				face.Tx() <- ndn.MakeData(packet.Interest, payload)
			}
		}
		log.Println("Closing RX")
	}()

	go func() {
		defer wg.Done()

		var tick <-chan time.Time
		if *transmit > 0 {
			tick = time.Tick(*transmit)
		}

		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGINT)

		for {
			select {
			case t := <-tick:
				name := fmt.Sprintf("%s/%d", *prefix, t.UnixNano())
				face.Tx() <- ndn.MakeInterest(name, ndn.MustBeFreshFlag)
			case <-interrupt:
				goto STOP
			}
		}

	STOP:
		close(face.Tx())
		log.Println("Closing TX")
	}()

	wg.Wait()
	cleanup()
}
