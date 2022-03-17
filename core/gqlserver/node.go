package gqlserver

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"reflect"

	"github.com/graphql-go/graphql"
)

var (
	idKey      [64]byte
	idEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)
	nodeTypes  = map[string]*NodeType{}

	//lint:ignore ST1005 'Node' is a proper noun referring to GraphQL type
	errNotFound   = errors.New("Node not found")
	errNoRetrieve = errors.New("cannot retrieve Node")
	errNoDelete   = errors.New("cannot delete Node")
	errWrongType  = errors.New("ID refers to wrong NodeType")
)

func xorID(value []byte) []byte {
	for i, b := range value {
		value[i] = b ^ idKey[i%len(idKey)]
	}
	return value
}

func makeID(prefix string, suffix interface{}) (id string) {
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

// NodeType defines a Node subtype.
type NodeType struct {
	prefix string
	typ    reflect.Type

	object *graphql.Object

	// GetID extracts unprefixed ID from the source object.
	GetID func(source interface{}) string

	// Retrieve fetches an object from unprefixed ID.
	Retrieve func(id string) (interface{}, error)

	// Delete deletes the source object.
	Delete func(source interface{}) error
}

// Annotate updates ObjectConfig with Node interface and "id" field.
//
// The 'id' can be resolved from:
//  - nt.GetID function.
//  - ObjectConfig 'nid' field, which must have NonNullID, NonNullInt, or NonNullString type.
// If neither is present, this function panics.
func (nt *NodeType) Annotate(oc graphql.ObjectConfig) graphql.ObjectConfig {
	if oc.Interfaces == nil {
		oc.Interfaces = []*graphql.Interface{}
	}
	oc.Interfaces = append(oc.Interfaces.([]*graphql.Interface), nodeInterface)

	if oc.Fields == nil {
		oc.Fields = graphql.Fields{}
	}
	fields := oc.Fields.(graphql.Fields)

	var resolve graphql.FieldResolveFn
	if nt.GetID != nil {
		resolve = func(p graphql.ResolveParams) (interface{}, error) {
			return makeID(nt.prefix, nt.GetID(p.Source)), nil
		}
	} else if nidField := fields["nid"]; nidField != nil {
		switch nidField.Type {
		case NonNullID, NonNullInt, NonNullString:
			resolve = func(p graphql.ResolveParams) (interface{}, error) {
				nid, e := nidField.Resolve(p)
				if e != nil {
					return nil, e
				}
				return makeID(nt.prefix, nid), nil
			}
		}
	}
	if resolve == nil {
		panic("cannot resolve 'id' field")
	}

	fields["id"] = &graphql.Field{
		Type:        graphql.NewNonNull(graphql.ID),
		Description: "Globally unique ID.",
		Resolve:     resolve,
	}

	return oc
}

// Register enables accessing Node of this type by ID.
func (nt *NodeType) Register(object *graphql.Object) {
	nt.object = object
	Schema.Types = append(Schema.Types, object)

	if nodeTypes[nt.prefix] != nil {
		panic("duplicate prefix " + nt.prefix)
	}
	nodeTypes[nt.prefix] = nt
}

// NewNodeType creates a NodeType.
func NewNodeType(value interface{}) *NodeType {
	return NewNodeTypeNamed("", value)
}

// NewNodeTypeNamed creates a NodeType with specified name.
func NewNodeTypeNamed(name string, value interface{}) (nt *NodeType) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		if elem := typ.Elem(); elem.Kind() == reflect.Interface {
			typ = elem
		}
	}
	if name == "" {
		name = typ.String()
	}

	nt = &NodeType{
		prefix: name,
		typ:    typ,
	}
	return nt
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
				return nt.object
			}
		}
		return nil
	},
})

// RetrieveNode locates Node by full ID.
func RetrieveNode(id interface{}) (*NodeType, interface{}, error) {
	prefix, suffix, ok := parseID(id.(string))
	if !ok {
		return nil, nil, errNotFound
	}

	nt := nodeTypes[prefix]
	if nt == nil || nt.Retrieve == nil {
		return nt, nil, errNoRetrieve
	}

	obj, e := nt.Retrieve(suffix)
	if e != nil {
		return nt, nil, e
	}
	if val := reflect.ValueOf(obj); obj == nil || (val.Kind() == reflect.Ptr && val.IsNil()) {
		return nt, nil, errNotFound
	}
	return nt, obj, nil
}

// RetrieveNodeOfType locates Node by full ID, ensures it has correct type, and assigns it to *ptr.
func RetrieveNodeOfType(expectedNodeType *NodeType, id, ptr interface{}) error {
	nt, node, e := RetrieveNode(id)
	if e != nil {
		return e
	}
	if nt != expectedNodeType {
		return errWrongType
	}
	reflect.ValueOf(ptr).Elem().Set(reflect.ValueOf(node))
	return nil
}

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
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, obj, e := RetrieveNode(p.Args["id"])
			return obj, e
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
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			nt, obj, e := RetrieveNode(p.Args["id"])
			if e != nil || obj == nil {
				return false, e
			}

			if nt.Delete == nil {
				return true, errNoDelete
			}
			return true, nt.Delete(obj)
		},
	})
}
