package appinit

import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/socketface"
)

var theFaceTable iface.FaceTable

func GetFaceTable() iface.FaceTable {
	if theFaceTable.GetPtr() == nil {
		theFaceTable = iface.NewFaceTable()
	}
	return theFaceTable
}

// Queue capacity configuration for new faces.
var (
	ETHFACE_RXQ_CAPACITY    = 64
	ETHFACE_TXQ_CAPACITY    = 64
	SOCKETFACE_RXQ_CAPACITY = 512
	SOCKETFACE_TXQ_CAPACITY = 64
)

// Create face by FaceUri and add to the FaceTable.
func NewFaceFromUri(u faceuri.FaceUri) (face *iface.Face, e error) {
	create := newFaceByScheme[u.Scheme]
	if create == nil {
		return nil, fmt.Errorf("cannot create face with scheme %s", u.Scheme)
	}
	face, e = create(u)
	if e == nil {
		GetFaceTable().SetFace(*face)
	}
	return face, e
}

// Functions to create face by FaceUri for each FaceUri scheme.
// These functions do not add face to the FaceTable.
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
	return newEthFaceFromDev(port)
}

// Create face on DPDK device and add to the FaceTable.
func NewFaceFromEthDev(port dpdk.EthDev) (face *iface.Face, e error) {
	if !port.IsValid() {
		return nil, errors.New("DPDK device is invalid")
	}
	face, e = newEthFaceFromDev(port)
	if e == nil {
		GetFaceTable().SetFace(*face)
	}
	return face, e
}

func newEthFaceFromDev(port dpdk.EthDev) (*iface.Face, error) {
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
	return &face.Face, nil
}

func newSocketFace(u faceuri.FaceUri) (face *iface.Face, e error) {
	network, address := u.Scheme[:3], u.Host

	var conn net.Conn
	if network == "udp" {
		raddr, e := net.ResolveUDPAddr(network, address)
		if e != nil {
			return nil, fmt.Errorf("net.ResolveUDPAddr(%s,%s): %v", network, address, e)
		}
		var laddr net.UDPAddr
		laddr.Port = raddr.Port
		conn, e = net.DialUDP(network, &laddr, raddr)
	} else {
		conn, e = net.Dial(network, address)
	}
	if e != nil {
		return nil, fmt.Errorf("net.Dial(%s,%s): %v", network, address, e)
	}

	var cfg socketface.Config
	cfg.Mempools = makeFaceMempools(dpdk.NUMA_SOCKET_ANY)
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, dpdk.NUMA_SOCKET_ANY)
	cfg.RxqCapacity = SOCKETFACE_RXQ_CAPACITY
	cfg.TxqCapacity = SOCKETFACE_TXQ_CAPACITY

	face = &socketface.New(conn, cfg).Face
	return face, nil
}

func makeFaceMempools(socket dpdk.NumaSocket) (mempools iface.Mempools) {
	mempools.IndirectMp = MakePktmbufPool(MP_IND, socket)
	mempools.NameMp = MakePktmbufPool(MP_NAME, socket)
	mempools.HeaderMp = MakePktmbufPool(MP_ETHTX, socket)
	return mempools
}

// Create RxLooper for one face.
func MakeRxLooper(face iface.Face) iface.IRxLooper {
	faceId := face.GetFaceId()
	switch faceId.GetKind() {
	case iface.FaceKind_EthDev:
		return ethface.EthFace{face}
	case iface.FaceKind_Socket:
		return socketface.NewRxGroup(socketface.Get(faceId))
	}
	return nil
}

// Create TxLooper for one face.
func MakeTxLooper(face iface.Face) iface.ITxLooper {
	return iface.NewTxLooper(face)
}
