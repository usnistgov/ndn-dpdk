package createface

import (
	"errors"
	"fmt"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/ndn"
)

type CreateArg struct {
	Remote *faceuri.FaceUri
	Local  *faceuri.FaceUri
}

func Create(args ...CreateArg) (faces []iface.IFace, e error) {
	if !isInitialized {
		return nil, errors.New("facecreate package is uninitialized")
	}

	ctx := newCreateContext(len(args))
	for i, arg := range args {
		if e = ctx.Add(i, arg); e != nil {
			return nil, fmt.Errorf("arg[%d]: %v", i, e)
		}
	}
	if e = ctx.Launch(); e != nil {
		return nil, e
	}
	return ctx.Faces, nil
}

type createContext struct {
	Faces   []iface.IFace
	eth     map[dpdk.EthDev]*createContextEth
	sockRxg *socketface.RxGroup
	hasMock bool
}

type createContextEth struct {
	ethface.PortConfig
	multicastIndex int
	unicastIndex   []int
}

func newCreateContext(count int) (ctx createContext) {
	ctx.Faces = make([]iface.IFace, count)
	ctx.eth = make(map[dpdk.EthDev]*createContextEth)
	return ctx
}

func (ctx *createContext) Add(i int, arg CreateArg) error {
	if arg.Remote == nil {
		return errors.New("remote FaceUri is missing")
	}
	if arg.Local != nil && arg.Remote.Scheme != arg.Local.Scheme {
		return errors.New("local scheme differs from remote scheme")
	}

	switch arg.Remote.Scheme {
	case "ether":
		return ctx.addEth(i, arg)
	case "mock":
		return ctx.addMock(i)
	}
	return ctx.addSock(i, arg)
}

func findEthDev(devName string) dpdk.EthDev {
	for _, ethdev := range dpdk.ListEthDevs() {
		if faceuri.CleanEthdevName(ethdev.GetName()) == devName {
			return ethdev
		}
	}
	return dpdk.ETHDEV_INVALID
}

func (ctx *createContext) addEth(i int, arg CreateArg) error {
	if !theConfig.EnableEth {
		return errors.New("Ethernet face feature is disabled")
	}
	if arg.Local == nil {
		return errors.New("local FaceUri is missing")
	}
	devName, remote, vid := arg.Remote.ExtractEther()
	_, local, _ := arg.Local.ExtractEther()
	if vid != 0 {
		return errors.New("VLAN is not implemented")
	}
	if faceuri.MacAddress(local).IsGroupAddress() {
		return errors.New("local MAC address must be unicast")
	}
	isMulticast := faceuri.MacAddress(remote).IsGroupAddress()
	if isMulticast && remote.String() != ndn.GetEtherMcastAddr().String() {
		return errors.New("remote MAC address must be either unicast or well-known NDN multicast group")
	}

	ethdev := findEthDev(devName)
	if ethdev == dpdk.ETHDEV_INVALID {
		return errors.New("EthDev not found")
	}
	if ethface.FindPort(ethdev) != nil {
		return errors.New("EthDev is already active")
	}

	ectx := ctx.eth[ethdev]
	if ectx == nil {
		ectx = new(createContextEth)
		ectx.EthDev = ethdev
		ectx.Local = local
		ectx.multicastIndex = -1
		ctx.eth[ethdev] = ectx
	}

	if ectx.Local.String() != local.String() {
		return errors.New("conflicting local MAC address")
	}
	if isMulticast {
		if ectx.Multicast {
			return errors.New("EthDev already has multicast face")
		}
		ectx.Multicast = true
		ectx.multicastIndex = i
	} else {
		ectx.Unicast = append(ectx.Unicast, remote)
		ectx.unicastIndex = append(ectx.unicastIndex, i)
	}

	return nil
}

func (ctx *createContext) addSock(i int, arg CreateArg) (e error) {
	if !theConfig.EnableSock {
		return errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(-1, dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	cfg.RxqCapacity = theConfig.SockRxqFrames
	cfg.TxqCapacity = theConfig.SockTxqFrames

	face, e := socketface.NewFromUri(arg.Remote, arg.Local, cfg)
	if e != nil {
		return e
	}
	ctx.Faces[i] = face
	return startSockRxtx(face)
}

var hasMockFaces = false

func (ctx *createContext) addMock(i int) (e error) {
	if !theConfig.EnableMock {
		return errors.New("mock face feature is disabled")
	}
	if !hasMockFaces {
		if mockface.FaceMempools, e = theCallbacks.CreateFaceMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
			return e
		}
		hasMockFaces = true
	}

	face := mockface.New()
	ctx.Faces[i] = face
	return startMockRxtx(face)
}

func (ctx *createContext) Launch() error {
	for ethdev, ectx := range ctx.eth {
		e := ctx.launchEth(ectx)
		if e != nil {
			return fmt.Errorf("eth[%s]: %v", ethdev.GetName(), e)
		}
	}
	return nil
}

func (ctx *createContext) launchEth(ectx *createContextEth) (e error) {
	numaSocket := ectx.EthDev.GetNumaSocket()

	cfg := ectx.PortConfig
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(numaSocket); e != nil {
		return e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(-1, numaSocket); e != nil {
		return e
	}
	cfg.RxqCapacity = theConfig.EthRxqFrames
	cfg.TxqCapacity = theConfig.EthTxqFrames

	port, e := ethface.NewPort(cfg)
	if e != nil {
		return e
	}

	if e := startEthRxtx(port); e != nil {
		return e
	}

	if ectx.multicastIndex >= 0 {
		face := port.GetMulticastFace()
		ctx.Faces[ectx.multicastIndex] = face
	}
	for i, face := range port.ListUnicastFaces() {
		ctx.Faces[ectx.unicastIndex[i]] = face
	}
	return nil
}
