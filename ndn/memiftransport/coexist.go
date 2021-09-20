package memiftransport

import "errors"

type coexistEntry struct {
	Role Role
	IDs  map[int]bool
}

// CoexistMap determines whether two memif transports can coexist.
type CoexistMap struct {
	m map[string]*coexistEntry
}

// Has determines whether there's an existing transport with the same socketName.
func (c CoexistMap) Has(socketName string) bool {
	return c.m[socketName] != nil
}

// Check determines whether creating a transport from given locator would conflict with existing transports.
// loc.ApplyDefaults() should have been called.
func (c CoexistMap) Check(loc Locator) error {
	entry := c.m[loc.SocketName]
	if entry == nil {
		return nil
	}
	if entry.Role != loc.Role {
		return errors.New("duplicate SocketName with different role")
	}
	if entry.IDs[loc.ID] {
		return errors.New("duplicate SocketName+ID")
	}
	return nil
}

// Add inserts a transport.
func (c *CoexistMap) Add(loc Locator) {
	entry := c.m[loc.SocketName]
	if entry == nil {
		entry = &coexistEntry{
			Role: loc.Role,
			IDs:  map[int]bool{},
		}
		c.m[loc.SocketName] = entry
	}
	entry.IDs[loc.ID] = true
}

// Remove deletes a transport.
func (c *CoexistMap) Remove(loc Locator) {
	entry := c.m[loc.SocketName]
	delete(entry.IDs, loc.ID)
	if len(entry.IDs) == 0 {
		delete(c.m, loc.SocketName)
	}
}

// NewCoexistMap creates an empty CoexistMap.
func NewCoexistMap() CoexistMap {
	return CoexistMap{
		m: map[string]*coexistEntry{},
	}
}
