// Package uintalloc allocates numeric identifiers absent in a map.
package uintalloc

import "math/rand/v2"

func hasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

// Alloc allocates uint32 or uint64 identifier.
//
//	m: map keyed by identifier.
//	rng: random number generator.
func Alloc[K uint32 | uint64, V any, M ~map[K]V](m M, rng func() K) (key K) {
	for key == 0 || hasKey(m, key) {
		key = rng()
	}
	return
}

// Alloc32 allocates uint32 identifier.
func Alloc32[V any, M ~map[uint32]V](m M) (key uint32) {
	return Alloc(m, rand.Uint32)
}

// Alloc64 allocates uint64 identifier.
func Alloc64[V any, M ~map[uint64]V](m M) (key uint64) {
	return Alloc(m, rand.Uint64)
}
