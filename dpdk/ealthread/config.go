package ealthread

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rickb777/plural"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var (
	lcoresPlural = plural.FromOne("%d lcore", "%d lcores")
	nArePlural   = plural.FromZero("none is", "only %d is", "only %d are")
)

// Config contains lcore allocation config.
//
// All roles needed by the application must be specified.
type Config map[string]RoleConfig

// ValidateRoles ensures the configured roles match application roles and minimums.
func (c Config) ValidateRoles(roles map[string]int) error {
	errs := []error{}
	for role, min := range roles {
		if n := c[role].Count(); n < min {
			errs = append(errs, fmt.Errorf("role %s needs at least %s but %s configured",
				role, lcoresPlural.FormatInt(min), nArePlural.FormatInt(n)))
		}
	}

	for role := range c {
		if _, ok := roles[role]; !ok {
			errs = append(errs, fmt.Errorf("unknown role %s", role))
		}
	}

	return errors.Join(errs...)
}

// Extract moves specified roles to another Config, and validates their minimums.
func (c Config) Extract(roles map[string]int) (p Config, e error) {
	p = Config{}
	if c == nil {
		return p, nil
	}

	for role, rc := range c {
		if _, ok := roles[role]; ok {
			p[role] = rc
		}
	}
	if e = p.ValidateRoles(roles); e != nil {
		return nil, e
	}

	for role := range roles {
		delete(c, role)
	}
	return p, nil
}

func (c Config) assignWorkers(filter eal.LCorePredicate) (m map[string]eal.LCores, e error) {
	m = map[string]eal.LCores{}
	errs := []error{}

	workersByID := map[int]eal.LCore{}
	for _, worker := range eal.Workers {
		if filter(worker) {
			workersByID[worker.ID()] = worker
		}
	}
	for role, rc := range c {
		for _, core := range rc.LCores {
			worker, ok := workersByID[core]
			if ok {
				m[role] = append(m[role], worker)
				delete(workersByID, worker.ID())
			} else {
				errs = append(errs, fmt.Errorf("lcore %d for role %s does not exist", core, role))
			}
		}
	}

	var unusedWorkers eal.LCores
	for _, worker := range workersByID {
		unusedWorkers = append(unusedWorkers, worker)
	}
	workersBySocket := unusedWorkers.ByNumaSocket()
	for role, rc := range c {
		for socketID, count := range rc.PerNuma {
			socket := eal.NumaSocketFromID(socketID)
			if avail := len(workersBySocket[socket]); avail >= count {
				m[role] = append(m[role], workersBySocket[socket][:count]...)
				workersBySocket[socket] = workersBySocket[socket][count:]
			} else {
				errs = append(errs, fmt.Errorf("role %s has %s configured on NUMA socket %d but %s available after prior assignments",
					role, lcoresPlural.FormatInt(count), socketID, nArePlural.FormatInt(avail)))
			}
		}
	}

	return m, errors.Join(errs...)
}

// RoleConfig contains lcore allocation config for a role.
//
// In JSON, it should be either:
//   - a list of lcore IDs, which cannot overlap with other roles.
//   - an object, where each key is a NUMA socket and each value is the number of lcores on this socket.
type RoleConfig struct {
	LCores  []int
	PerNuma map[int]int
}

// Count returns the number of lcores configured.
func (c RoleConfig) Count() (sum int) {
	sum = len(c.LCores)
	for _, n := range c.PerNuma {
		sum += n
	}
	return sum
}

// MarshalJSON implements json.Marshaler interface.
func (c RoleConfig) MarshalJSON() ([]byte, error) {
	switch {
	case len(c.LCores) > 0:
		return json.Marshal(c.LCores)
	case len(c.PerNuma) > 0:
		return json.Marshal(c.PerNuma)
	default:
		return []byte(`[]`), nil
	}
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (c *RoleConfig) UnmarshalJSON(j []byte) error {
	*c = RoleConfig{}

	e0 := json.Unmarshal(j, &c.LCores)
	if e0 == nil {
		return nil
	}

	e1 := json.Unmarshal(j, &c.PerNuma)
	if e1 == nil {
		return nil
	}

	return fmt.Errorf("decode as lcore list: %w\ndecode as socket=>count map: %w", e0, e1)
}
