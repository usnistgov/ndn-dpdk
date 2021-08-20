package memiftransport

import "errors"

// CoexistEntry is an entry in CoexistMap.
type CoexistEntry struct {
	Role Role
	IDs  map[int]bool
}

// CoexistMap determines whether two memif transports can coexist.
type CoexistMap map[string]*CoexistEntry

// Check determines whether creating a transport from given locator would conflict with existing transports.
// loc.ApplyDefaults() should have been called.
func (m CoexistMap) Check(loc Locator) error {
	entry := m[loc.SocketName]
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
func (m CoexistMap) Add(loc Locator) {
	entry := m[loc.SocketName]
	if entry == nil {
		entry = &CoexistEntry{
			Role: loc.Role,
			IDs:  map[int]bool{},
		}
		m[loc.SocketName] = entry
	}
	entry.IDs[loc.ID] = true
}

// Remove deletes a transport.
func (m CoexistMap) Remove(loc Locator) {
	entry := m[loc.SocketName]
	delete(entry.IDs, loc.ID)
	if len(entry.IDs) == 0 {
		delete(m, loc.SocketName)
	}
}
