package appinit

/*
#include <rte_config.h>
*/
import "C"
import (
	"fmt"
	"strings"
	"sync"

	"github.com/jaypipes/ghw"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

type MempoolConfig struct {
	Capacity     int
	PrivSize     int
	DataroomSize int
}

var (
	mempoolEnableNuma = false
	mempoolDecideNuma sync.Once
	mempoolCfgs       = make(map[string]*MempoolConfig)
	mempools          = make(map[string]dpdk.PktmbufPool)
)

// Register template for mempool creation.
func RegisterMempool(key string, cfg MempoolConfig) {
	if _, ok := mempoolCfgs[key]; ok {
		log.Panicf("RegisterPktmbufPool(%s) duplicate", key)
	}

	if strings.ContainsRune(key, '#') {
		log.Panicf("RegisterPktmbufPool(%s) key cannot contain '#'", key)
	}

	mempoolCfgs[key] = &cfg
}

// Modify mempool capacity in template.
func ConfigureMempool(key string, capacity int) {
	cfg, ok := mempoolCfgs[key]
	if !ok {
		log.Panicf("ConfigurePktmbufPool(%s) unregistered", key)
	}

	cfg.Capacity = capacity
}

type MempoolCapacityConfig struct {
	Capacity     int
	DataroomSize int
}

// Init config section for mempool capacity.
type MempoolsCapacityConfig map[string]MempoolCapacityConfig

func (cfg MempoolsCapacityConfig) Apply() {
	for key, entry := range cfg {
		tpl, ok := mempoolCfgs[key]
		if !ok {
			log.WithField("key", key).Warn("unknown mempool template")
			continue
		}
		tpl.Capacity = entry.Capacity
		if entry.DataroomSize > 0 {
			if entry.DataroomSize < tpl.DataroomSize {
				log.WithFields(makeLogFields(
					"key", key, "oldDataroom", tpl.DataroomSize,
					"newDataroom", entry.DataroomSize)).Info("decreasing dataroom size")
			}
			tpl.DataroomSize = entry.DataroomSize
		}
	}
}

// Get or create a mempool on specified NumaSocket.
func MakePktmbufPool(key string, socket dpdk.NumaSocket) dpdk.PktmbufPool {
	mempoolDecideNuma.Do(func() {
		topology, e := ghw.Topology()
		if e != nil {
			return
		}
		mempoolEnableNuma = len(topology.Nodes) > 1
	})

	logEntry := log.WithField("template", key)

	cfg, ok := mempoolCfgs[key]
	if !ok {
		logEntry.Panic("mempool template unregistered")
	}

	if cfg.Capacity <= 0 {
		logEntry.Fatal("mempool bad config: capacity must be positive")
	}
	if ((cfg.Capacity + 1) & cfg.Capacity) != 0 {
		logEntry.Warn("mempool nonoptimal config: capacity is not 2^q-1")
	}

	useSocket := socket
	if !mempoolEnableNuma {
		useSocket = dpdk.NUMA_SOCKET_ANY
	}
	name := fmt.Sprintf("%s#%d", key, useSocket)
	logEntry = logEntry.WithFields(makeLogFields("name", name, "socket", socket, "use-socket", useSocket))
	if mp, ok := mempools[name]; ok {
		logEntry.Debug("mempool found")
		return mp
	}

	mp, e := dpdk.NewPktmbufPool(name, cfg.Capacity, cfg.PrivSize, cfg.DataroomSize, useSocket)
	if e != nil {
		logEntry.WithError(e).Fatal("mempool creation failed")
	}
	mempools[name] = mp
	logEntry.Debug("mempool created")
	return mp
}

// Registered mempool templates.
const (
	MP_IND   = "IND"   // indirect mbufs
	MP_ETHRX = "ETHRX" // RX Ethernet frames
	MP_NAME  = "NAME"  // name linearize
	MP_HDR   = "HDR"   // TX Ethernet+NDNLP+Interest headers
	MP_INTG  = "INTG"  // modifying Interest guiders
	MP_INT   = "INT"   // TX Ethernet+NDNLP and encoding Interest
	MP_DATA0 = "DATA0" // TX Ethernet+NDNLP+Data name prefix
	MP_DATA1 = "DATA1" // TX Data name suffix and payload
)

var SizeofEthLpHeaders = ethface.SizeofTxHeader

func init() {
	RegisterMempool(MP_IND,
		MempoolConfig{
			Capacity:     1048575,
			PrivSize:     0,
			DataroomSize: 0,
		})
	RegisterMempool(MP_ETHRX,
		MempoolConfig{
			Capacity:     524287,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: 2560, // >= MTU+sizeof(rte_ether_hdr)
		})
	RegisterMempool(MP_NAME,
		MempoolConfig{
			Capacity:     65535,
			PrivSize:     0,
			DataroomSize: ndn.NAME_MAX_LENGTH,
		})
	RegisterMempool(MP_HDR,
		MempoolConfig{
			Capacity:     65535,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: SizeofEthLpHeaders() + ndn.Interest_Headroom,
		})
	RegisterMempool(MP_INTG,
		MempoolConfig{
			Capacity:     65535,
			PrivSize:     0,
			DataroomSize: ndn.Interest_SizeofGuider,
		})
	RegisterMempool(MP_INT,
		MempoolConfig{
			Capacity:     65535,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: SizeofEthLpHeaders() + ndn.Interest_Headroom + ndn.Interest_TailroomMax,
		})
	RegisterMempool(MP_DATA0,
		MempoolConfig{
			Capacity:     65535,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: dpdk.MBUF_DEFAULT_HEADROOM + ndn.DataGen_GetTailroom0(ndn.NAME_MAX_LENGTH),
		})
	RegisterMempool(MP_DATA1,
		MempoolConfig{
			Capacity:     255,
			PrivSize:     0,
			DataroomSize: dpdk.MBUF_DEFAULT_HEADROOM + ndn.DataGen_GetTailroom1(ndn.NAME_MAX_LENGTH, 1500),
		})
}

// Provide mempools to createface package.
// This should be called after createface.Config has been applied.
func ProvideCreateFaceMempools() {
	numaSockets := make(map[dpdk.NumaSocket]bool)
	for _, numaSocket := range createface.ListRxTxNumaSockets() {
		numaSockets[numaSocket] = true
	}
	if len(numaSockets) == 0 {
		numaSockets[dpdk.NUMA_SOCKET_ANY] = true
	} else if len(numaSockets) > 1 && numaSockets[dpdk.NUMA_SOCKET_ANY] {
		delete(numaSockets, dpdk.NUMA_SOCKET_ANY)
	}
	for numaSocket := range numaSockets {
		createface.AddMempools(numaSocket,
			MakePktmbufPool(MP_ETHRX, numaSocket),
			iface.Mempools{
				IndirectMp: MakePktmbufPool(MP_IND, numaSocket),
				NameMp:     MakePktmbufPool(MP_NAME, numaSocket),
				HeaderMp:   MakePktmbufPool(MP_HDR, numaSocket),
			})
	}
}
