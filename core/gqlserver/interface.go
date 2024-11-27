package gqlserver

import (
	"reflect"
	"slices"

	"github.com/graphql-go/graphql"
)

// FieldDefToField converts FieldDefinition to *Field.
func FieldDefToField(d *graphql.FieldDefinition) *graphql.Field {
	return &graphql.Field{
		Name:              d.Name,
		Description:       d.Description,
		Type:              d.Type,
		Resolve:           d.Resolve,
		Subscribe:         d.Subscribe,
		DeprecationReason: d.DeprecationReason,
	}
}

// Interface defines a GraphQL interface.
type Interface struct {
	Interface *graphql.Interface
	types     map[reflect.Type]*graphql.Object
}

// AppendTo appends this interface to oc.Interfaces slice.
func (it *Interface) AppendTo(oc *graphql.ObjectConfig) {
	interfaces, _ := oc.Interfaces.([]*graphql.Interface)
	if slices.Index(interfaces, it.Interface) < 0 {
		oc.Interfaces = append(interfaces, it.Interface)
	}
}

// CopyFieldsTo adds fields of this interface into InterfaceConfig.Fields or ObjectConfig.Fields.
// If a field of same name exists at the destination, only Type and Description are overwritten.
// This may be used for inheritance between interfaces or for sharing implementation.
func (it *Interface) CopyFieldsTo(fieldsAny any) graphql.Fields {
	fields, ok := fieldsAny.(graphql.Fields)
	if !ok {
		fields = graphql.Fields{}
	}

	for name, field := range it.Interface.Fields() {
		if f, ok := fields[name]; ok {
			if f.Type == nil {
				f.Type = field.Type
			}
			if f.Description == "" {
				f.Description = field.Description
			}
			continue
		}
		fields[name] = FieldDefToField(field)
	}
	return fields
}

func (it *Interface) resolveType(p graphql.ResolveTypeParams) *graphql.Object {
	typ := reflect.TypeOf(p.Value)
	if ot, ok := it.types[typ]; ok {
		return ot
	}

	for t, ot := range it.types {
		if typ.AssignableTo(t) {
			return ot
		}
	}
	return nil
}

// NewInterface creates a GraphQL interface.
// ic.Fields should contain necessary fields.
// ic.ResolveType will be overwritten.
func NewInterface(ic graphql.InterfaceConfig) (it *Interface) {
	it = &Interface{
		types: map[reflect.Type]*graphql.Object{},
	}
	ic.ResolveType = it.resolveType
	it.Interface = graphql.NewInterface(ic)
	return it
}

// ImplementsInterface records an object implementing an interface.
// This also appends the object to Schema.Types to ensure that it appears in the schema.
func ImplementsInterface[T any](ot *graphql.Object, it *Interface) {
	it.types[reflect.TypeFor[T]()] = ot
	Schema.Types = append(Schema.Types, ot)
}
