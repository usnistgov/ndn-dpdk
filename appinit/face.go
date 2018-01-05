package appinit

import (
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/socketface"
)

var FACE_RXQ_CAPACITY = 64 // RX queue capacity for new faces
var FACE_TXQ_CAPACITY = 64 // TX queue capacity for new faces

func NewFaceFromUri(u faceuri.FaceUri) (*iface.Face, error) {
	create := newFaceByScheme[u.Scheme]
	if create == nil {
		return nil, fmt.Errorf("cannot create face with scheme %s", u.Scheme)
	}
	return create(u)
}

var newFaceByScheme = map[string]func(u faceuri.FaceUri) (*iface.Face, error){
	"dev":  newEthFace,
	"udp4": newSocketFace,
	"tcp4": newSocketFace,
}

func newEthFace(u faceuri.FaceUri) (*iface.Face, error) {
	port := dpdk.FindEthDev(u.Host)
	if !port.IsValid() {
		return nil, fmt.Errorf("DPDK device %s not found", u.Host)
	}

	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: FACE_RXQ_CAPACITY,
		Socket: port.GetNumaSocket(),
		Mp:     MakePktmbufPool(MP_ETHRX, port.GetNumaSocket())})
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: FACE_TXQ_CAPACITY,
		Socket: port.GetNumaSocket()})
	_, _, e := port.Configure(cfg)
	if e != nil {
		return nil, fmt.Errorf("port(%d).Configure: %v", port, e)
	}

	port.SetPromiscuous(true)

	e = port.Start()
	if e != nil {
		return nil, fmt.Errorf("port(%d).Start: %v", port, e)
	}

	face, e := ethface.New(port,
		MakePktmbufPool(MP_IND, port.GetNumaSocket()),
		MakePktmbufPool(MP_ETHTX, port.GetNumaSocket()))
	if e != nil {
		return nil, fmt.Errorf("ethface.New(%d): %v", port, e)
	}

	return &face.Face, nil
}

func newSocketFace(u faceuri.FaceUri) (*iface.Face, error) {
	network, address := u.Scheme[:3], u.Host

	conn, e := net.Dial(network, address)
	if e != nil {
		return nil, fmt.Errorf("net.Dial(%s,%s): %v", network, address, e)
	}

	var cfg socketface.Config
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, dpdk.NUMA_SOCKET_ANY)
	cfg.RxqCapacity = FACE_RXQ_CAPACITY
	cfg.TxIndirectMp = MakePktmbufPool(MP_IND, dpdk.NUMA_SOCKET_ANY)
	cfg.TxHeaderMp = MakePktmbufPool(MP_ETHTX, dpdk.NUMA_SOCKET_ANY)
	cfg.TxqCapacity = FACE_TXQ_CAPACITY

	face := socketface.New(conn, cfg)
	return &face.Face, nil
}
