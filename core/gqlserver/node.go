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
	errWrongType  = errors.New("ID refers to wrong NodeType")
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

// RetrieveNode locates Node by full ID.
func RetrieveNode(id interface{}) (*NodeType, interface{}, error) {
	idDecoded, e := base64.RawURLEncoding.DecodeString(id.(string))
	if e != nil {
		return nil, nil, nil
	}
	tokens := strings.SplitN(string(idDecoded), ":", 2)
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

// RetrieveNodeOfType locates Node by full ID, and ensures it has correct type.
func RetrieveNodeOfType(expectedNodeType *NodeType, id interface{}) (interface{}, error) {
	nt, node, e := RetrieveNode(id)
	if e != nil || node == nil {
		return nil, e
	}
	if nt != expectedNodeType {
		return nil, errWrongType
	}
	return node, nil
}

func init() {
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
