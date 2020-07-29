package gqlserver

import (
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
)

// NodeResolve is a function that fetches an object from ID.
//  id: bare ID without prefix.
type NodeResolve func(id string) (interface{}, error)

// NodeType contains helpers for a node type.
type NodeType string

// Annotate updates ObjectConfig with "id" field and Node interface.
func (nt NodeType) Annotate(object *graphql.ObjectConfig, getID func(source interface{}) string) {
	prefix := string(nt)

	if object.Fields == nil {
		object.Fields = graphql.Fields{}
	}
	fields := object.Fields.(graphql.Fields)
	fields["id"] = &graphql.Field{
		Type:        graphql.NewNonNull(graphql.ID),
		Description: "Globally unique ID.",
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return prefix + ":" + getID(p.Source), nil
		},
	}

	if object.Interfaces == nil {
		object.Interfaces = []*graphql.Interface{}
	}
	object.Interfaces = append(object.Interfaces.([]*graphql.Interface), nodeInterface)
}

// Register registers the NodeType.
func (nt NodeType) Register(object *graphql.Object, value interface{}, resolve NodeResolve) {
	prefix := string(nt)
	if nodeResolves[prefix] != nil {
		panic("duplicate prefix " + prefix)
	}

	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		if elem := typ.Elem(); elem.Kind() == reflect.Interface {
			typ = elem
		}
	}
	if nodeObjectTypes[typ] != nil {
		panic("duplicate type " + typ.String())
	}

	nodeResolves[prefix] = resolve
	nodeObjectTypes[typ] = object
	Schema.Types = append(Schema.Types, object)
}

var nodeInterface = graphql.NewInterface(graphql.InterfaceConfig{
	Name: "Node",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.ID),
		},
	},
	ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
		typ := reflect.TypeOf(p.Value)
		for t, object := range nodeObjectTypes {
			if typ.AssignableTo(t) {
				return object
			}
		}
		return nil
	},
})

var nodeResolves = make(map[string]NodeResolve)
var nodeObjectTypes = make(map[reflect.Type]*graphql.Object)

func init() {
	AddQuery(&graphql.Field{
		Name:        "node",
		Description: "Retrieve object by global ID.",
		Type:        nodeInterface,
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(graphql.ID),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			id := p.Args["id"].(string)
			tokens := strings.SplitN(id, ":", 2)
			resolve := nodeResolves[tokens[0]]
			if resolve == nil {
				return nil, nil
			}
			return resolve(tokens[1])
		},
	})
}
