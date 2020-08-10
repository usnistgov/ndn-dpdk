package gqlserver

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
)

var (
	nodeTypes = make(map[string]*NodeType)

	errNoRetrieve = errors.New("cannot retrieve Node")
	errNoDelete   = errors.New("cannot delete Node")
)

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
//  - ObjectConfig 'nid' field of NonNullInt or NonNullString type.
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
			return nt.makeID(nt.GetID(p.Source))
		}
	} else if nidField := fields["nid"]; nidField != nil {
		switch nidField.Type {
		case NonNullID, NonNullInt, NonNullString:
			resolve = func(p graphql.ResolveParams) (interface{}, error) {
				nid, e := nidField.Resolve(p)
				if e != nil {
					return nil, e
				}
				return nt.makeID(nid)
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

func (nt *NodeType) makeID(suffix interface{}) (interface{}, error) {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s:%v", nt.prefix, suffix)
	return base64.RawURLEncoding.EncodeToString(buffer.Bytes()), nil
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

func retrieveNode(p graphql.ResolveParams) (*NodeType, interface{}, error) {
	id, e := base64.RawURLEncoding.DecodeString(p.Args["id"].(string))
	if e != nil {
		return nil, nil, nil
	}
	tokens := strings.SplitN(string(id), ":", 2)
	if len(tokens) != 2 {
		return nil, nil, nil
	}

	nt := nodeTypes[tokens[0]]
	if nt == nil || nt.Retrieve == nil {
		return nt, nil, errNoRetrieve
	}

	obj, e := nt.Retrieve(tokens[1])
	if val := reflect.ValueOf(obj); obj == nil || (val.Kind() == reflect.Ptr && val.IsNil()) {
		obj = nil
	}
	return nt, obj, e
}

func init() {
	AddQuery(&graphql.Field{
		Name:        "node",
		Description: "Retrieve object by global ID.",
		Type:        nodeInterface,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: NonNullID,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			_, obj, e := retrieveNode(p)
			return obj, e
		},
	})

	AddMutation(&graphql.Field{
		Name:        "delete",
		Description: "Delete object by global ID. The result indicates whether the object previously exists.",
		Type:        graphql.Boolean,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: NonNullID,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			nt, obj, e := retrieveNode(p)
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
