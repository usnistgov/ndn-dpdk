package appinit

import (
	"errors"
	"fmt"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

type FaceQueueCapacityConfig struct {
	EthRxFrames int
	EthTxPkts   int
	EthTxFrames int

	SocketRxFrames int
	SocketTxPkts   int
	SocketTxFrames int
}

func (cfg FaceQueueCapacityConfig) Apply() {
	TheFaceQueueCapacityConfig = cfg
}

var TheFaceQueueCapacityConfig FaceQueueCapacityConfig

func init() {
	TheFaceQueueCapacityConfig.EthRxFrames = 256
	TheFaceQueueCapacityConfig.EthTxPkts = 256
	TheFaceQueueCapacityConfig.EthTxFrames = 256
	TheFaceQueueCapacityConfig.SocketRxFrames = 256
	TheFaceQueueCapacityConfig.SocketTxPkts = 256
	TheFaceQueueCapacityConfig.SocketTxFrames = 256
}

// Create face by FaceUri.
func NewFaceFromUri(remote, local *faceuri.FaceUri) (face iface.IFace, e error) {
	if remote == nil {
		return nil, errors.New("remote FaceUri is empty")
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
	"ether": newEthFace,
	"udp4":  newSocketFace,
	"tcp4":  newSocketFace,
	"mock":  newMockFace,
}

func newEthFace(remote, local *faceuri.FaceUri) (iface.IFace, error) {
	hostname := remote.Hostname()
	for _, ethdev := range dpdk.ListEthDevs() {
		if faceuri.CleanEthdevName(ethdev.GetName()) == hostname {
			return newEthFaceFromDev(ethdev, 1)
		}
	}
	return nil, fmt.Errorf("DPDK device %s not found", hostname)
}

// Create face on DPDK device.
func NewFaceFromEthDev(ethdev dpdk.EthDev, nRxThreads int) (face iface.IFace, e error) {
	if !ethdev.IsValid() {
		return nil, errors.New("DPDK device is invalid")
	}
	face, e = newEthFaceFromDev(ethdev, nRxThreads)
	return face, e
}

func newEthFaceFromDev(ethdev dpdk.EthDev, nRxThreads int) (iface.IFace, error) {
	numaSocket := ethdev.GetNumaSocket()
	var cfg ethface.PortConfig
	cfg.Mempools = makeFaceMempools(numaSocket)
	cfg.EthDev = ethdev
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, numaSocket)
	cfg.RxqCapacity = TheFaceQueueCapacityConfig.EthRxFrames
	cfg.TxqCapacity = TheFaceQueueCapacityConfig.EthTxFrames
	cfg.Local = nil
	cfg.Multicast = true

	port, e := ethface.NewPort(cfg)
	if e != nil {
		return nil, e
	}

	return port.GetMulticastFace(), nil
	// XXX nRxThreads is ignored
}

func newSocketFace(remote, local *faceuri.FaceUri) (face iface.IFace, e error) {
	cfg := NewSocketFaceCfg(dpdk.NUMA_SOCKET_ANY)
	return socketface.NewFromUri(remote, local, cfg)
}

func NewSocketFaceCfg(socket dpdk.NumaSocket) (cfg socketface.Config) {
	cfg.Mempools = makeFaceMempools(socket)
	cfg.RxMp = MakePktmbufPool(MP_ETHRX, socket)
	cfg.RxqCapacity = TheFaceQueueCapacityConfig.SocketRxFrames
	cfg.TxqCapacity = TheFaceQueueCapacityConfig.SocketTxFrames
	return cfg
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
		rxl := ethface.NewRxLoop(1, face.GetNumaSocket())
		if e := rxl.AddPort(face.(*ethface.EthFace).GetPort()); e != nil {
			return nil
		}
		return rxl
	case iface.FaceKind_Socket:
		return socketface.NewRxGroup(face.(*socketface.SocketFace))
	}
	return nil
}

// Create TxLooper for one face.
func MakeTxLooper(face iface.IFace) iface.ITxLooper {
	return iface.NewSingleTxLoop(face)
}
