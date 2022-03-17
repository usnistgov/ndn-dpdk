package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"strings"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var templates = map[string]*template{}

func validateTemplateID(id string) bool {
	for _, ch := range id {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", ch) {
			return false
		}
	}
	return true
}

// PoolInfo augments *Pool with NUMA socket information.
type PoolInfo struct {
	*Pool
	socket eal.NumaSocket
}

// NumaSocket returns NUMA socket on which the Pool was created.
func (pool PoolInfo) NumaSocket() eal.NumaSocket {
	return pool.socket
}

// Template represents a template to create mempools.
type Template interface {
	// ID returns template identifier.
	ID() string

	// Config returns current configuration.
	Config() PoolConfig

	// Update changes mempool configuration.
	// PrivSize can only be increased.
	// Dataroom can be updated only if original dataroom is non-zero.
	// Returns self.
	Update(update PoolConfig) Template

	// Pools returns a list of created Pools.
	Pools() []PoolInfo

	// Get retrieves or creates a Pool on the given NUMA socket.
	// Errors are fatal.
	Get(socket eal.NumaSocket) *Pool
}

type template struct {
	id    string
	cfg   PoolConfig
	pools map[eal.NumaSocket]*Pool
}

func (tpl *template) ID() string {
	return tpl.id
}

func (tpl *template) Config() PoolConfig {
	return tpl.cfg
}

func (tpl *template) Update(update PoolConfig) Template {
	if update.Capacity > 0 {
		tpl.cfg.Capacity = update.Capacity
	}

	if update.PrivSize > tpl.cfg.PrivSize {
		tpl.cfg.PrivSize = update.PrivSize
	} else if update.PrivSize > 0 {
		logger.Info("ignoring attempt to decrease PrivSize",
			zap.String("key", tpl.id),
			zap.Int("oldPrivSize", tpl.cfg.PrivSize),
			zap.Int("newPrivSize", update.PrivSize),
		)
	}

	if tpl.cfg.Dataroom > 0 && update.Dataroom > 0 {
		if update.Dataroom < tpl.cfg.Dataroom {
			logger.Info("decreasing Dataroom",
				zap.String("key", tpl.id),
				zap.Int("oldDataroom", tpl.cfg.Dataroom),
				zap.Int("newDataroom", update.Dataroom),
			)
		}
		tpl.cfg.Dataroom = update.Dataroom
	}

	return tpl
}

func (tpl *template) Pools() (list []PoolInfo) {
	for socket, pool := range tpl.pools {
		list = append(list, PoolInfo{Pool: pool, socket: socket})
	}
	return list
}

func (tpl *template) Get(socket eal.NumaSocket) *Pool {
	logEntry := logger.With(zap.String("template", tpl.id))

	useSocket := socket
	if len(eal.Sockets) <= 1 {
		useSocket = eal.NumaSocket{}
	}
	logEntry = logEntry.With(
		socket.ZapField("socket"),
		useSocket.ZapField("use-socket"),
		zap.Any("cfg", tpl.cfg),
	)

	if pool, ok := tpl.pools[useSocket]; ok {
		logEntry.Debug("mempool found",
			zap.Stringer("pool", pool),
		)
		return pool
	}

	pool, e := NewPool(tpl.cfg, useSocket)
	if e != nil {
		logEntry.Fatal("mempool creation failed", zap.Error(e))
	}
	tpl.pools[useSocket] = pool
	logEntry.Debug("mempool created",
		zap.Stringer("pool", pool),
	)
	return pool
}

// RegisterTemplate adds a mempool template.
func RegisterTemplate(id string, cfg PoolConfig) Template {
	logEntry := logger.With(zap.String("template", id))

	if _, ok := templates[id]; ok {
		logEntry.Panic("duplicate template ID")
	}
	if !validateTemplateID(id) {
		logEntry.Panic("template ID can only contain upper-case letters and digits")
	}
	tpl := &template{
		id:    id,
		cfg:   cfg,
		pools: map[eal.NumaSocket]*Pool{},
	}
	templates[id] = tpl
	return tpl
}

// FindTemplate locates template by ID.
func FindTemplate(id string) Template {
	tpl, ok := templates[id]
	if !ok {
		return nil
	}
	return tpl
}

// Predefined mempool templates.
var (
	// Direct is a mempool template for direct mbufs.
	Direct Template

	// Indirect is a mempool template for indirect mbufs.
	Indirect Template
)

func init() {
	Direct = RegisterTemplate("DIRECT", PoolConfig{
		Capacity: 524287,
		PrivSize: 0,
		Dataroom: C.RTE_MBUF_DEFAULT_BUF_SIZE,
	})

	Indirect = RegisterTemplate("INDIRECT", PoolConfig{
		Capacity: 1048575,
	})
}

// TemplateUpdates contains updates to several mempool templates.
type TemplateUpdates map[string]PoolConfig

// Apply applies the updates.
func (updates TemplateUpdates) Apply() {
	for key, update := range updates {
		tpl := FindTemplate(key)
		if tpl == nil {
			logger.Warn("unknown mempool template",
				zap.String("key", key),
			)
			continue
		}
		tpl.Update(update)
	}
}
