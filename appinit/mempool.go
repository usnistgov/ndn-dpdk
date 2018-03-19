package appinit

/*
#include <rte_config.h>
*/
import "C"
import (
	"fmt"
	"log"
	"math"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

type MempoolConfig struct {
	Capacity     int
	CacheSize    int
	PrivSize     uint16
	DataRoomSize uint16
}

var mempoolCfgs = make(map[string]*MempoolConfig)
var mempools = make(map[string]dpdk.PktmbufPool)

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
func ConfigureMempool(key string, capacity int, cacheSize int) {
	cfg, ok := mempoolCfgs[key]
	if !ok {
		log.Panicf("ConfigurePktmbufPool(%s) unregistered", key)
	}

	cfg.Capacity = capacity
	cfg.CacheSize = cacheSize
}

// Get or create a mempool on specified NumaSocket.
func MakePktmbufPool(key string, socket dpdk.NumaSocket) dpdk.PktmbufPool {
	cfg, ok := mempoolCfgs[key]
	if !ok {
		log.Panicf("MakePktmbufPool(%s) unregistered", key)
	}

	if cfg.Capacity <= 0 {
		Exitf(EXIT_BAD_CONFIG, "MakePktmbufPool(%s) bad config: capacity must be positive", key)
	}
	if ((cfg.Capacity + 1) & cfg.Capacity) != 0 {
		log.Printf("MakePktmbufPool(%s) nonoptimal config: capacity is not 2^q-1", key)
	}
	maxCacheSize := int(math.Min(float64(int(C.RTE_MEMPOOL_CACHE_MAX_SIZE)),
		float64(cfg.Capacity)/1.5))
	if cfg.CacheSize < 0 || cfg.CacheSize > maxCacheSize {
		Exitf(EXIT_BAD_CONFIG, "MakePktmbufPool(%s) bad config: cache size must be between 0 and %d",
			key, maxCacheSize)
	}
	if cfg.CacheSize > 0 && cfg.Capacity%cfg.CacheSize != 0 {
		log.Printf("MakePktmbufPool(%s) nonoptimal config: capacity is not a multiply of cacheSize",
			key)
	}

	name := fmt.Sprintf("%s#%d", key, socket)
	if mp, ok := mempools[name]; ok {
		return mp
	}

	mp, e := dpdk.NewPktmbufPool(name, cfg.Capacity, cfg.CacheSize,
		cfg.PrivSize, cfg.DataRoomSize, socket)
	if e != nil {
		Exitf(EXIT_MEMPOOL_INIT_ERROR, "MakePktmbufPool(%s,%d): %v", key, socket, e)
	}
	mempools[name] = mp
	return mp
}

// Registered mempool templates.
const (
	MP_IND   = "__IND"   // indirect mbufs
	MP_ETHRX = "__ETHRX" // RX Ethernet frames
	MP_NAME  = "__NAME"  // name linearize
	MP_HDR   = "__HDR"   // TX Ethernet+NDNLP+Interest headers
	MP_INTG  = "__INTG"  // modifying Interest guiders
	MP_INT   = "__INT"   // encoding Interest
	MP_DATA1 = "__DATA1" // encoding Data header
	MP_DATA2 = "__DATA2" // encoding Data signature
)

func init() {
	RegisterMempool(MP_IND,
		MempoolConfig{
			Capacity:     262143,
			CacheSize:    219,
			PrivSize:     0,
			DataRoomSize: 0,
		})
	RegisterMempool(MP_ETHRX,
		MempoolConfig{
			Capacity:     262143,
			CacheSize:    219,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataRoomSize: 9014, // MTU+sizeof(ether_hdr)
		})
	RegisterMempool(MP_NAME,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataRoomSize: ndn.NAME_MAX_LENGTH,
		})
	RegisterMempool(MP_HDR,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataRoomSize: ethface.SizeofHeaderMempoolDataRoom() + uint16(ndn.EncodeInterest_GetHeadroom()),
		})
	RegisterMempool(MP_INTG,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataRoomSize: uint16(ndn.ModifyInterest_SizeofGuider()),
		})
	RegisterMempool(MP_INT,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataRoomSize: uint16(ndn.EncodeInterest_GetHeadroom() + ndn.EncodeInterest_GetTailroomMax()),
		})
	RegisterMempool(MP_DATA1,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataRoomSize: uint16(ndn.EncodeData1_GetHeadroom() + ndn.EncodeData1_GetTailroomMax()),
		})
	RegisterMempool(MP_DATA2,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataRoomSize: uint16(ndn.EncodeData2_GetHeadroom() + ndn.EncodeData2_GetTailroom()),
		})
}
