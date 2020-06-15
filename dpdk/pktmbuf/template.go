package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"fmt"
	"strings"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Template represents a template to create mempools.
type Template struct {
	key string
}

// GetConfig returns current PoolConfig of the template.
func (tpl Template) GetConfig() PoolConfig {
	return *templateConfigs[tpl.key]
}

var (
	templateConfigs = make(map[string]*PoolConfig)
	templatePools   = make(map[string]*Pool)
)

const templateKeyChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func validateTemplateKey(key string) bool {
	for _, ch := range key {
		if !strings.ContainsRune(templateKeyChars, ch) {
			return false
		}
	}
	return true
}

// RegisterTemplate adds a mempool template.
func RegisterTemplate(key string, cfg PoolConfig) Template {
	if _, ok := templateConfigs[key]; ok {
		log.Panicf("RegisterTemplate(%s) duplicate", key)
	}
	if !validateTemplateKey(key) {
		log.Panicf("RegisterTemplate(%s) key can only contain upper-case letters and digits", key)
	}
	templateConfigs[key] = &cfg
	return Template{key}
}

// FindTemplate locates template by key.
func FindTemplate(key string) *Template {
	if _, ok := templateConfigs[key]; ok {
		return &Template{key}
	}
	return nil
}

// Update changes mempool configuration.
// PrivSize can only be increased.
// Dataroom can be updated only if original dataroom is non-zero.
func (tpl Template) Update(update PoolConfig) Template {
	cfg := templateConfigs[tpl.key]

	if update.Capacity > 0 {
		cfg.Capacity = update.Capacity
	}

	if update.PrivSize > cfg.PrivSize {
		cfg.PrivSize = update.PrivSize
	} else if update.PrivSize > 0 {
		log.WithFields(makeLogFields(
			"key", tpl.key, "oldPrivSize", cfg.PrivSize,
			"newPrivSize", update.PrivSize)).Info("ignoring attempt to decrease PrivSize")
	}

	if cfg.Dataroom > 0 && update.Dataroom > 0 {
		if update.Dataroom < cfg.Dataroom {
			log.WithFields(makeLogFields(
				"key", tpl.key, "oldDataroom", cfg.Dataroom,
				"newDataroom", update.Dataroom)).Info("decreasing Dataroom")
		}
		cfg.Dataroom = update.Dataroom
	}

	return tpl
}

// MakePool creates or retrieves a Pool, preferably on given NUMA socket.
// Failure causes panic or fatal error.
func (tpl Template) MakePool(socket eal.NumaSocket) *Pool {
	logEntry := log.WithField("template", tpl.key)
	cfg := templateConfigs[tpl.key]

	useSocket := socket
	if len(eal.ListNumaSockets()) <= 1 {
		useSocket = eal.NumaSocket{}
	}
	name := fmt.Sprintf("%s#%s", tpl.key, useSocket)
	logEntry = logEntry.WithFields(makeLogFields("name", name, "socket", socket, "use-socket", useSocket, "cfg", *cfg))

	if mp, ok := templatePools[name]; ok {
		logEntry.Debug("mempool found")
		return mp
	}

	mp, e := NewPool(name, *cfg, useSocket)
	if e != nil {
		logEntry.WithError(e).Fatal("mempool creation failed")
	}
	templatePools[name] = mp
	logEntry.Debug("mempool created")
	return mp
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
			log.WithField("key", key).Warn("unknown mempool template")
			continue
		}
		tpl.Update(update)
	}
}
