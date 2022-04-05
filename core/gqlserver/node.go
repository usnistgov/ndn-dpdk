package gqlserver

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/graphql-go/graphql"
)

var (
	idKey      [64]byte
	idEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)
	nodeTypes  = map[string]*NodeTypeBase{}

	errNotFound = errors.New("node not found")
	errNoDelete = errors.New("cannot delete node")
)

func xorID(value []byte) []byte {
	for i, b := range value {
		value[i] = b ^ idKey[i%len(idKey)]
	}
	return value
}

func makeID(prefix string, suffix any) (id string) {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s:%v", prefix, suffix)
	return idEncoding.EncodeToString(xorID(buffer.Bytes()))
}

func parseID(id string) (prefix, suffix string, ok bool) {
	value, e := idEncoding.DecodeString(id)
	if e != nil {
		return
	}
	value = xorID(value)

	tokens := bytes.SplitN(value, []byte{':'}, 2)
	if len(tokens) != 2 {
		return
	}

	return string(tokens[0]), string(tokens[1]), true
}

// NodeTypeBase contains non-generic fields of NodeType.
type NodeTypeBase struct {
	Object      *graphql.Object
	prefix      string
	typ         reflect.Type
	retrieveAny func(suffix string) (node any, e error)
	deleteByID  func(suffix string) (ok bool, e error)
}

// NodeType defines a Node subtype.
type NodeType[T any] struct {
	NodeTypeBase
	retrieve func(suffix string) T
}

// Retrieve retrieves node by ID.
func (nt *NodeType[T]) Retrieve(id string) (node T) {
	prefix, suffix, ok := parseID(id)
	if !ok || prefix != nt.prefix {
		return node
	}
	return nt.retrieve(suffix)
}

// NewNodeType creates a NodeType.
func NewNodeType[T any](oc graphql.ObjectConfig, nc NodeConfig[T]) (nt *NodeType[T]) {
	if oc.Interfaces == nil {
		oc.Interfaces = []*graphql.Interface{}
	}
	oc.Interfaces = append(oc.Interfaces.([]*graphql.Interface), nodeInterface)

	if oc.Fields == nil {
		oc.Fields = graphql.Fields{}
	}
	fields := oc.Fields.(graphql.Fields)
	fields["id"] = &graphql.Field{
		Type:        graphql.NewNonNull(graphql.ID),
		Description: "Globally unique ID.",
		Resolve:     nc.makeResolveID(oc.Name, fields),
	}

	obj := graphql.NewObject(oc)
	Schema.Types = append(Schema.Types, obj)

	nt = &NodeType[T]{
		NodeTypeBase: NodeTypeBase{
			Object: obj,
			prefix: oc.Name,
			typ:    reflect.TypeOf((*T)(nil)).Elem(),
		},
		retrieve: nc.makeRetrieve(),
	}
	nt.retrieveAny = func(suffix string) (any, error) {
		node := nt.retrieve(suffix)
		if reflect.ValueOf(node).IsZero() {
			return nil, errNotFound
		}
		return node, nil
	}
	nt.deleteByID = func(suffix string) (ok bool, e error) {
		node := nt.retrieve(suffix)
		if reflect.ValueOf(node).IsZero() {
			return false, nil
		}
		if nc.Delete == nil {
			return true, errNoDelete
		}
		return true, nc.Delete(node)
	}

	nodeTypes[nt.prefix] = &nt.NodeTypeBase
	return nt
}

// NodeConfig contains options to construct a NodeType.
type NodeConfig[T any] struct {
	// GetID extracts un-prefixed ID from the source object.
	GetID func(source T) string

	// Retrieve fetches an object from un-prefixed ID.
	// Returning zero value indicates the object does not exist.
	Retrieve func(id string) T

	// RetrieveInt fetches an object from un-prefixed ID parsed as integer.
	RetrieveInt func(id int) T

	// Delete deletes the source object.
	Delete func(source T) error
}

func (nc NodeConfig[T]) makeResolveID(prefix string, fields graphql.Fields) graphql.FieldResolveFn {
	if nc.GetID != nil {
		return func(p graphql.ResolveParams) (any, error) {
			return makeID(prefix, nc.GetID(p.Source.(T))), nil
		}
	}

	if nidField := fields["nid"]; nidField != nil {
		switch nidField.Type {
		case NonNullID, NonNullInt, NonNullString:
			return func(p graphql.ResolveParams) (any, error) {
				nid, e := nidField.Resolve(p)
				if e != nil {
					return nil, e
				}
				return makeID(prefix, nid), nil
			}
		}
	}

	logger.Panic("cannot resolve 'id' field")
	return nil
}

func (nc NodeConfig[T]) makeRetrieve() func(suffix string) T {
	switch {
	case nc.Retrieve != nil:
		return nc.Retrieve
	case nc.RetrieveInt != nil:
		return func(suffix string) (node T) {
			n, e := strconv.Atoi(suffix)
			if e != nil {
				return
			}
			return nc.RetrieveInt(n)
		}
	}
	logger.Panic("either Retrieve or RetrieveInt must be set")
	return nil
}

var nodeInterface = graphql.NewInterface(graphql.InterfaceConfig{
	Name: "Node",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: NonNullID,
		},
	},
	ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
		typ := reflect.TypeOf(p.Value)
		for _, nt := range nodeTypes {
			if typ.AssignableTo(nt.typ) {
				return nt.Object
			}
		}
		return nil
	},
})

func init() {
	if _, e := rand.Read(idKey[:]); e != nil {
		panic(e)
	}

	AddQuery(&graphql.Field{
		Name:        "node",
		Description: "Retrieve object by global ID.",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: NonNullID,
			},
		},
		Type: nodeInterface,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			prefix, suffix, ok := parseID(p.Args["id"].(string))
			if !ok {
				return nil, nil
			}
			if nt := nodeTypes[prefix]; nt != nil {
				return nt.retrieveAny(suffix)
			}
			return nil, nil
		},
	})

	AddMutation(&graphql.Field{
		Name:        "delete",
		Description: "Delete object by global ID. The result indicates whether the object previously exists.",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: NonNullID,
			},
		},
		Type: NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			prefix, suffix, ok := parseID(p.Args["id"].(string))
			if !ok {
				return false, nil
			}
			if nt := nodeTypes[prefix]; nt != nil {
				return nt.deleteByID(suffix)
			}
			return false, nil
		},
	})
}
