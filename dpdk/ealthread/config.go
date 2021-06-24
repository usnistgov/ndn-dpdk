package ealthread

import (
	"encoding/json"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/multierr"
)

// Config contains lcore allocation config.
//
// All roles needed by the application must be specified.
type Config map[string]RoleConfig

// ValidateRoles ensures the configured roles match application roles and minimums.
func (c Config) ValidateRoles(roles map[string]int) error {
	ok := len(c) == len(roles)
	if ok {
		for role, min := range roles {
			rc, found := c[role]
			ok = ok && found && rc.Count() >= min
		}
	}
	if ok {
		return nil
	}

	cRoles := []string{}
	for role := range c {
		cRoles = append(cRoles, role)
	}
	return fmt.Errorf("configured roles [%v] do not match application roles [%v]", cRoles, roles)
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
				errs = append(errs, fmt.Errorf("role %s needs %d lcores on NUMA socket %d but only %d are available", role, count, socketID, avail))
			}
		}
	}

	return m, multierr.Combine(errs...)
}

// RoleConfig contains lcore allocation config for a role.
//
// In JSON, it should be either:
//  - a list of lcore IDs, which cannot overlap with other roles.
//  - an object, where each key is a NUMA socket and each value is the number of lcores on this socket.
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

	return multierr.Append(
		fmt.Errorf("decode as lcore list: %w", e0),
		fmt.Errorf("decode as socket=>count map: %w", e1),
	)
}
