package gqlsingleton

import (
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// Singleton represents a single instance GraphQL object.
type Singleton struct {
	sync.Mutex
	id    int
	value any
}

// Get returns the object.
func (s *Singleton) Get() any {
	s.Lock()
	defer s.Unlock()
	return s.value
}

// SetNodeType assigns callback functions on NodeType.
// The source object must implement io.Closer interface.
func (s *Singleton) SetNodeType(nt *gqlserver.NodeType) {
	nt.GetID = func(source any) string {
		s.Lock()
		defer s.Unlock()
		if source == s.value {
			return strconv.Itoa(s.id)
		}
		return ""
	}
	nt.Retrieve = func(id string) (any, error) {
		s.Lock()
		defer s.Unlock()
		if id == strconv.Itoa(s.id) {
			return s.value, nil
		}
		return nil, nil
	}
	nt.Delete = func(source any) error {
		closer := source.(io.Closer)
		if e := closer.Close(); e != nil {
			return e
		}

		s.Lock()
		defer s.Unlock()
		if source == s.value {
			s.id++
			s.value = nil
		}
		return nil
	}
}

// CreateWith wraps a create object mutation resolver with singleton lock.
func (s *Singleton) CreateWith(fn graphql.FieldResolveFn) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		s.Lock()
		defer s.Unlock()
		if s.value != nil {
			return nil, errors.New("object already exist")
		}
		value, e := fn(p)
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
func (s *Singleton) QueryList(p graphql.ResolveParams) (any, error) {
	s.Lock()
	defer s.Unlock()
	if s.value == nil {
		return []any{}, nil
	}
	return []any{s.value}, nil
}
