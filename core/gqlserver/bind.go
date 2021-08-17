package gqlserver

import (
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"go.uber.org/zap"
)

// FieldTypes contains known GraphQL types of fields.
type FieldTypes map[reflect.Type]graphql.Type

func (m FieldTypes) resolveType(typ reflect.Type) graphql.Type {
	if t := m[typ]; t != nil {
		if kind := typ.Kind(); kind == reflect.Ptr || kind == reflect.Slice {
			return t
		}
		return graphql.NewNonNull(graphql.GetNullable(t).(graphql.Type))
	}

	switch typ.Kind() {
	case reflect.Ptr:
		return graphql.GetNullable(m.resolveType(typ.Elem())).(graphql.Type)
	case reflect.Slice:
		return graphql.NewList(m.resolveType(typ.Elem()))
	case reflect.Array:
		return graphql.NewNonNull(graphql.NewList(m.resolveType(typ.Elem())))
	case reflect.Bool:
		return NonNullBoolean
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return NonNullInt
	case reflect.Float32, reflect.Float64:
		return graphql.Float // NaN is null
	case reflect.Uint64, reflect.Int64, reflect.String:
		return NonNullString
	}

	logger.Panic("FieldTypes cannot resolve type", zap.Stringer("type", typ))
	return nil
}

func (m FieldTypes) bindFields(typ reflect.Type, saveField func(name string, t graphql.Type, resolve graphql.FieldResolveFn)) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for _, field := range reflect.VisibleFields(typ) {
		if !field.IsExported() {
			continue
		}

		tag, ok := field.Tag.Lookup("json")
		if !ok {
			continue
		}
		tagTokens := strings.Split(tag, ",")

		name := tagTokens[0]
		if name == "-" {
			continue
		}

		ft := m.resolveType(field.Type)
		if len(tagTokens) >= 2 && tagTokens[1] == "omitempty" {
			ft = graphql.GetNullable(ft).(graphql.Type)
		}

		index := append([]int{}, field.Index...)
		saveField(name, ft, func(p graphql.ResolveParams) (interface{}, error) {
			obj := reflect.ValueOf(p.Source)
			for _, i := range index {
				if obj.Kind() == reflect.Ptr {
					if obj.IsNil() {
						return nil, nil
					}
					obj = obj.Elem()
				}

				obj = obj.Field(i)
				if !obj.IsValid() {
					return nil, nil
				}
			}
			return obj.Interface(), nil
		})
	}
}

// BindFields creates graphql.Field from a struct.
func BindFields(value interface{}, m FieldTypes) graphql.Fields {
	fields := graphql.Fields{}
	m.bindFields(reflect.TypeOf(value), func(name string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.Field{
			Type:    t,
			Resolve: resolve,
		}
	})
	return fields
}

// BindInputFields creates graphql.InputObjectConfigFieldMap from a struct.
func BindInputFields(value interface{}, m FieldTypes) graphql.InputObjectConfigFieldMap {
	fields := graphql.InputObjectConfigFieldMap{}
	m.bindFields(reflect.TypeOf(value), func(name string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.InputObjectFieldConfig{Type: t}
	})
	return fields
}
