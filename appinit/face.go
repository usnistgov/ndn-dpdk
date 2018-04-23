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
	SOCKETFACE_RXQ_CAPACITY = 128
	SOCKETFACE_TXQ_CAPACITY = 64
)

// Create face by FaceUri.
func NewFaceFromUri(remote, local *faceuri.FaceUri) (face iface.IFace, e error) {
	if remote == nil {
		return nil, errors.New("remote FaceUri is empty")
	}
	if local != nil && local.Scheme != remote.Scheme {
		return nil, errors.New("remote and local FaceUris have different schemes")
	}

	create := newFaceByScheme[remote.Scheme]
	if create == nil {
		return nil, fmt.Errorf("cannot create face with scheme %s", remote.Scheme)
	}
	face, e = create(remote, local)
	return face, e
}

// Functions to create face by FaceUri for each FaceUri scheme.
var newFaceByScheme = map[string]func(remote, local *faceuri.FaceUri) (iface.IFace, error){
	"dev":  newEthFace,
	"udp4": newSocketFace,
	"tcp4": newSocketFace,
	"mock": newMockFace,
}

func newEthFace(remote, local *faceuri.FaceUri) (iface.IFace, error) {
	if local != nil {
		return nil, errors.New("eth scheme does not accept local FaceUri")
	}

	port := dpdk.FindEthDev(remote.Host)
	if !port.IsValid() {
		return nil, fmt.Errorf("DPDK device %s not found", remote.Host)
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

func newSocketFace(remote, local *faceuri.FaceUri) (face iface.IFace, e error) {
	var conn net.Conn
	if remote.Scheme == "udp4" {
		raddr, e := net.ResolveUDPAddr(remote.Scheme, remote.Host)
		if e != nil {
			return nil, fmt.Errorf("net.ResolveUDPAddr(%s,%s): %v", remote.Scheme, remote.Host, e)
		}
		laddr := &net.UDPAddr{Port: raddr.Port}
		if local != nil {
			if laddr, e = net.ResolveUDPAddr(local.Scheme, local.Host); e != nil {
				return nil, fmt.Errorf("net.ResolveUDPAddr(%s,%s): %v", local.Scheme, local.Host, e)
			}
		}
		conn, e = net.DialUDP(remote.Scheme, laddr, raddr)
		if e != nil {
			return nil, fmt.Errorf("net.DialUDP(%s,%s,%s): %v", remote.Scheme, laddr, raddr, e)
		}
	} else if local != nil {
		return nil, fmt.Errorf("%s scheme does not accept local FaceUri", remote.Scheme)
	} else {
		conn, e = net.Dial(remote.Scheme, remote.Host)
		if e != nil {
			return nil, fmt.Errorf("net.Dial(%s,%s): %v", remote.Scheme, remote.Host, e)
		}
	}

	var cfg socketface.Config
	cfg.Mempools = makeFaceMempools(dpdk.NUMA_SOCKET_ANY)
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, dpdk.NUMA_SOCKET_ANY)
	cfg.RxqCapacity = SOCKETFACE_RXQ_CAPACITY
	cfg.TxqCapacity = SOCKETFACE_TXQ_CAPACITY
	return socketface.New(conn, cfg), nil
}

func newMockFace(remote, local *faceuri.FaceUri) (face iface.IFace, e error) {
	if local != nil {
		return nil, errors.New("mock scheme does not accept local FaceUri")
	}
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
