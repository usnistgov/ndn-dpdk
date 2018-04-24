package socketface_test

import (
	"net"
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/ndn"
)

var socketfaceCfg socketface.Config

func TestMain(m *testing.M) {
	socketfaceCfg = socketface.Config{
		Mempools: iface.Mempools{
			IndirectMp: dpdktestenv.MakeIndirectMp(4095),
			NameMp:     dpdktestenv.MakeMp("name", 4095, 0, ndn.NAME_MAX_LENGTH),
			HeaderMp:   dpdktestenv.MakeMp("header", 4095, 0, ndn.PrependLpHeader_GetHeadroom()),
		},
		RxMp:        dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000),
		RxqCapacity: 64,
		TxqCapacity: 64,
	}

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

// Create net.Conn from file descriptor.
func makeConnFromFd(fd int) net.Conn {
	file := os.NewFile(uintptr(fd), "")
	if file == nil {
		panic(fd)
	}
	defer file.Close()

	conn, e := net.FileConn(file)
	if e != nil {
		panic(e)
	}
	return conn
}
