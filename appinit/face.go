package appinit

import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

// Queue capacity configuration for new faces.
var (
	ETHFACE_RXQ_CAPACITY    = 64
	ETHFACE_TXQ_CAPACITY    = 64
	SOCKETFACE_RXQ_CAPACITY = 512
	SOCKETFACE_TXQ_CAPACITY = 64
)

// Create face by FaceUri.
func NewFaceFromUri(u faceuri.FaceUri) (face iface.IFace, e error) {
	create := newFaceByScheme[u.Scheme]
	if create == nil {
		return nil, fmt.Errorf("cannot create face with scheme %s", u.Scheme)
	}
	face, e = create(u)
	return face, e
}

// Functions to create face by FaceUri for each FaceUri scheme.
var newFaceByScheme = map[string]func(u faceuri.FaceUri) (iface.IFace, error){
	"dev":  newEthFace,
	"udp4": newSocketFace,
	"tcp4": newSocketFace,
	"mock": newMockFace,
}

func newEthFace(u faceuri.FaceUri) (iface.IFace, error) {
	port := dpdk.FindEthDev(u.Host)
	if !port.IsValid() {
		return nil, fmt.Errorf("DPDK device %s not found", u.Host)
	}
	return newEthFaceFromDev(port)
}

// Create face on DPDK device.
func NewFaceFromEthDev(port dpdk.EthDev) (face iface.IFace, e error) {
	if !port.IsValid() {
		return nil, errors.New("DPDK device is invalid")
	}
	face, e = newEthFaceFromDev(port)
	return face, e
}

func newEthFaceFromDev(port dpdk.EthDev) (iface.IFace, error) {
	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: ETHFACE_RXQ_CAPACITY,
		Socket: port.GetNumaSocket(),
		Mp:     MakePktmbufPool(MP_ETHRX, port.GetNumaSocket())})
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: ETHFACE_TXQ_CAPACITY,
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

	face, e := ethface.New(port, makeFaceMempools(port.GetNumaSocket()))
	if e != nil {
		return nil, fmt.Errorf("ethface.New(%d): %v", port, e)
	}
	return face, nil
}

func newSocketFace(u faceuri.FaceUri) (face iface.IFace, e error) {
	network, address := u.Scheme[:3], u.Host

	var conn net.Conn
	if network == "udp" {
		raddr, e := net.ResolveUDPAddr(network, address)
		if e != nil {
			return nil, fmt.Errorf("net.ResolveUDPAddr(%s,%s): %v", network, address, e)
		}
		laddr := net.UDPAddr{Port: raddr.Port}
		conn, e = net.DialUDP(network, &laddr, raddr)
		if e != nil {
			return nil, fmt.Errorf("net.DialUDP(%s,%s): %v", network, address, e)
		}
	} else {
		conn, e = net.Dial(network, address)
		if e != nil {
			return nil, fmt.Errorf("net.Dial(%s,%s): %v", network, address, e)
		}
	}

	var cfg socketface.Config
	cfg.Mempools = makeFaceMempools(dpdk.NUMA_SOCKET_ANY)
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, dpdk.NUMA_SOCKET_ANY)
	cfg.RxqCapacity = SOCKETFACE_RXQ_CAPACITY
	cfg.TxqCapacity = SOCKETFACE_TXQ_CAPACITY

	return socketface.New(conn, cfg), nil
}

func newMockFace(u faceuri.FaceUri) (face iface.IFace, e error) {
	mockface.FaceMempools = makeFaceMempools(dpdk.NUMA_SOCKET_ANY)
	return mockface.New(), nil
}

func makeFaceMempools(socket dpdk.NumaSocket) (mempools iface.Mempools) {
	mempools.IndirectMp = MakePktmbufPool(MP_IND, socket)
	mempools.NameMp = MakePktmbufPool(MP_NAME, socket)
	mempools.HeaderMp = MakePktmbufPool(MP_HDR, socket)
	return mempools
}

// Create RxLooper for one face.
func MakeRxLooper(face iface.IFace) iface.IRxLooper {
	faceId := face.GetFaceId()
	switch faceId.GetKind() {
	case iface.FaceKind_Mock:
		return mockface.TheRxLoop
	case iface.FaceKind_Eth:
		return face.(iface.IRxLooper)
	case iface.FaceKind_Socket:
		return socketface.NewRxGroup(face.(*socketface.SocketFace))
	}
	return nil
}

// Create TxLooper for one face.
func MakeTxLooper(face iface.IFace) iface.ITxLooper {
	return iface.NewSingleTxLoop(face)
}
