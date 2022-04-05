// Package gqlsingleton provides singleton objects in GraphQL server.
package gqlsingleton

import (
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// Instance is the type constraint of Singleton.
type Instance interface {
	comparable
	io.Closer
}

// Singleton represents a single instance GraphQL object.
type Singleton[T Instance] struct {
	sync.Mutex
	id    int
	value T
}

// Get returns the object.
func (s *Singleton[T]) Get() T {
	s.Lock()
	defer s.Unlock()
	return s.value
}

// NodeConfig returns NodeConfig with callback functions.
func (s *Singleton[T]) NodeConfig() (nc gqlserver.NodeConfig[T]) {
	nc.GetID = func(source T) string {
		s.Lock()
		defer s.Unlock()
		if source == s.value {
			return strconv.Itoa(s.id)
		}
		return ""
	}
	nc.RetrieveInt = func(id int) (value T) {
		s.Lock()
		defer s.Unlock()
		if id == s.id {
			return s.value
		}
		return
	}
	nc.Delete = func(source T) error {
		if e := source.Close(); e != nil {
			return e
		}

		s.Lock()
		defer s.Unlock()
		if source == s.value {
			s.id++
			var zero T
			s.value = zero
		}
		return nil
	}
	return
}

// CreateWith wraps a create object mutation resolver with singleton lock.
func (s *Singleton[T]) CreateWith(f func(p graphql.ResolveParams) (value T, e error)) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		s.Lock()
		defer s.Unlock()
		var zero T
		if s.value != zero {
			return nil, errors.New("object already exist")
		}
		value, e := f(p)
		if e != nil {
			return nil, e
		}
		s.value = value
		s.id++
		return value, nil
	}
}

// QueryList provides an object list query resolver.
// The return type is [T!]!.
func (s *Singleton[T]) QueryList(p graphql.ResolveParams) (any, error) {
	s.Lock()
	defer s.Unlock()
	var zero T
	if s.value == zero {
		return []any{}, nil
	}
	return []any{s.value}, nil
}
