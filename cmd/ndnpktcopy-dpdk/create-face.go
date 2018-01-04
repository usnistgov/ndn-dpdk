package main

import (
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/socketface"
)

var faceByUri = make(map[string]*iface.Face)

func createFaceFromUri(faceUri string) (face *iface.Face, isNew bool, e error) {
	u, e := faceuri.Parse(faceUri)
	if e != nil {
		return nil, false, e
	}
	faceUri = u.String()

	if face, ok := faceByUri[faceUri]; ok {
		return face, false, nil
	}

	create := createFaceByScheme[u.Scheme]
	if create == nil {
		return nil, true, fmt.Errorf("cannot create face with scheme %s", u.Scheme)
	}

	face, e = create(u)
	if e != nil {
		return nil, true, e
	}
	faceByUri[faceUri] = face
	return face, true, e
}

var createFaceByScheme = map[string]func(u *faceuri.FaceUri) (*iface.Face, error){
	"dev":  createEthFace,
	"udp4": createSocketFace,
	"tcp4": createSocketFace,
}

func createEthFace(u *faceuri.FaceUri) (*iface.Face, error) {
	port := dpdk.FindEthDev(u.Host)
	if !port.IsValid() {
		return nil, fmt.Errorf("DPDK device %s not found", u.Host)
	}

	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: RXQ_CAPACITY,
		Socket: port.GetNumaSocket(), Mp: mpRx})
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: TXQ_CAPACITY, Socket: port.GetNumaSocket()})
	_, _, e := port.Configure(cfg)
	if e != nil {
		return nil, fmt.Errorf("port(%d).Configure: %v", port, e)
	}

	port.SetPromiscuous(true)

	e = port.Start()
	if e != nil {
		return nil, fmt.Errorf("port(%d).Start: %v", port, e)
	}

	face, e := ethface.New(port, mpIndirect, mpTxHdr)
	if e != nil {
		return nil, fmt.Errorf("ethface.New(%d): %v", port, e)
	}

	return &face.Face, nil
}

func createSocketFace(u *faceuri.FaceUri) (*iface.Face, error) {
	network, address := u.Scheme[:3], u.Host

	conn, e := net.Dial(network, address)
	if e != nil {
		return nil, fmt.Errorf("net.Dial(%s,%s): %v", network, address, e)
	}

	var cfg socketface.Config
	cfg.RxMp = mpRx
	cfg.RxqCapacity = RXQ_CAPACITY
	cfg.TxIndirectMp = mpIndirect
	cfg.TxHeaderMp = mpTxHdr
	cfg.TxqCapacity = TXQ_CAPACITY

	face := socketface.New(conn, cfg)
	return &face.Face, nil
}
