package appinit

/*
#include <rte_config.h>
*/
import "C"
import (
	"fmt"
	"math"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

type MempoolConfig struct {
	Capacity     int
	CacheSize    int
	PrivSize     int
	DataroomSize int
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

type MempoolCapacityConfig struct {
	Capacity  int
	CacheSize int
}

// Init config section for mempool capacity.
type MempoolsCapacityConfig map[string]MempoolCapacityConfig

func (cfg MempoolsCapacityConfig) Apply() {
	for key, cfg1 := range cfg {
		tpl, ok := mempoolCfgs[key]
		if !ok {
			log.WithField("key", key).Warn("unknown mempool template")
			continue
		}
		tpl.Capacity = cfg1.Capacity
		tpl.CacheSize = cfg1.CacheSize
	}
}

// Get or create a mempool on specified NumaSocket.
func MakePktmbufPool(key string, socket dpdk.NumaSocket) dpdk.PktmbufPool {
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
	maxCacheSize := int(math.Min(float64(int(C.RTE_MEMPOOL_CACHE_MAX_SIZE)),
		float64(cfg.Capacity)/1.5))
	if cfg.CacheSize < 0 || cfg.CacheSize > maxCacheSize {
		logEntry.Fatalf("mempool bad config: cache size must be between 0 and %d", maxCacheSize)
	}
	if cfg.CacheSize > 0 && cfg.Capacity%cfg.CacheSize != 0 {
		logEntry.Warn("mempool nonoptimal config: capacity is not a multiply of cacheSize")
	}

	name := fmt.Sprintf("%s#%d", key, socket)
	logEntry = logEntry.WithFields(makeLogFields("name", name, "socket", socket))
	if mp, ok := mempools[name]; ok {
		logEntry.Debug("mempool found")
		return mp
	}

	mp, e := dpdk.NewPktmbufPool(name, cfg.Capacity, cfg.CacheSize,
		cfg.PrivSize, cfg.DataroomSize, socket)
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
	MP_DATA  = "DATA"  // TX Ethernet+NDNLP and encoding Data
)

var SizeofEthLpHeaders = ethface.SizeofTxHeader

func init() {
	RegisterMempool(MP_IND,
		MempoolConfig{
			Capacity:     2097151,
			CacheSize:    337,
			PrivSize:     0,
			DataroomSize: 0,
		})
	RegisterMempool(MP_ETHRX,
		MempoolConfig{
			Capacity:     2097151,
			CacheSize:    337,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: 2560, // >= MTU+sizeof(ether_hdr)
		})
	RegisterMempool(MP_NAME,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataroomSize: ndn.NAME_MAX_LENGTH,
		})
	RegisterMempool(MP_HDR,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataroomSize: SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom(),
		})
	RegisterMempool(MP_INTG,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataroomSize: ndn.ModifyInterest_SizeofGuider(),
		})
	RegisterMempool(MP_INT,
		MempoolConfig{
			Capacity:  65535,
			CacheSize: 255,
			PrivSize:  ndn.SizeofPacketPriv(),
			DataroomSize: SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom() +
				ndn.EncodeInterest_GetTailroomMax(),
		})
	RegisterMempool(MP_DATA,
		MempoolConfig{
			Capacity:  65535,
			CacheSize: 255,
			PrivSize:  ndn.SizeofPacketPriv(),
			DataroomSize: SizeofEthLpHeaders() + ndn.EncodeData_GetHeadroom() +
				ndn.EncodeData_GetTailroomMax(),
		})
}
