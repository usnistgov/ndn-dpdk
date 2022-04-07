package gqlserver

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"go.uber.org/zap"
)

func makeFieldIndexResolver(index []int) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		r, e := reflect.Indirect(reflect.ValueOf(p.Source)).FieldByIndexErr(index)
		if e != nil {
			return nil, nil
		}
		return r.Interface(), nil
	}
}

// FieldTypes contains known GraphQL types of fields.
type FieldTypes map[reflect.Type]graphql.Type

func (m FieldTypes) resolveType(typ reflect.Type) graphql.Type {
	if t := m[typ]; t != nil {
		if kind := typ.Kind(); kind == reflect.Pointer || kind == reflect.Slice {
			return t
		}
		return toNonNull(t)
	}

	switch typ.Kind() {
	case reflect.Pointer:
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
	case reflect.Uint64:
		return NonNullUint64
	case reflect.Int64:
		return NonNullInt64
	case reflect.Float32, reflect.Float64:
		// NaN is null, so this would not allow NaN
		return graphql.NewNonNull(graphql.Float)
	case reflect.String:
		return NonNullString
	}

	logger.Panic("FieldTypes cannot resolve type", zap.Stringer("type", typ))
	return nil
}

func (m FieldTypes) bindFields(zero any, save func(name string, p fieldInfo)) {
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	for _, field := range reflect.VisibleFields(typ) {
		if !field.IsExported() {
			continue
		}

		jsonTag, ok := field.Tag.Lookup("json")
		if !ok || jsonTag == "-" {
			continue
		}
		jsonTokens := strings.Split(jsonTag, ",")
		name := jsonTokens[0]

		p := fieldInfo{
			Description: field.Tag.Get("gqldesc"),
			Index:       field.Index,
		}

		if dfltTag, ok := field.Tag.Lookup("gqldflt"); ok {
			dfltPtr := reflect.New(field.Type)
			if e := json.Unmarshal([]byte(dfltTag), dfltPtr.Interface()); e != nil {
				logger.Panic("cannot parse gqldflt",
					zap.String("field", field.Name),
					zap.Error(e),
				)
			}
			p.Default = dfltPtr.Elem().Interface()
		}

		p.Type = m.resolveType(field.Type)
		if p.Default != nil || (len(jsonTokens) >= 2 && jsonTokens[1] == "omitempty") {
			p.Type = graphql.GetNullable(p.Type).(graphql.Type)
		}

		save(name, p)
	}
}

type fieldInfo struct {
	Description string
	Default     any
	Type        graphql.Type
	Index       []int
}

func bindFieldsGeneric[T any, M ~map[string]*F, F any](m FieldTypes, convert func(fieldInfo) *F) M {
	fields := M{}
	var zero T
	m.bindFields(zero, func(name string, p fieldInfo) {
		fields[name] = convert(p)
	})
	return fields
}

// BindFields creates graphql.Fields from a struct.
// Field resolvers can accept either T or *T as source object.
func BindFields[T any](m FieldTypes) graphql.Fields {
	return bindFieldsGeneric[T, graphql.Fields](m, func(p fieldInfo) *graphql.Field {
		return &graphql.Field{
			Description: p.Description,
			Type:        p.Type,
			Resolve:     makeFieldIndexResolver(p.Index),
		}
	})
}

// BindInputFields creates graphql.InputObjectConfigFieldMap from a struct.
func BindInputFields[T any](m FieldTypes) graphql.InputObjectConfigFieldMap {
	return bindFieldsGeneric[T, graphql.InputObjectConfigFieldMap](m, func(p fieldInfo) *graphql.InputObjectFieldConfig {
		return &graphql.InputObjectFieldConfig{
			Description:  p.Description,
			Type:         p.Type,
			DefaultValue: p.Default,
		}
	})
}

// BindArguments creates graphql.FieldConfigArgument from a struct.
func BindArguments[T any](m FieldTypes) graphql.FieldConfigArgument {
	return bindFieldsGeneric[T, graphql.FieldConfigArgument](m, func(p fieldInfo) *graphql.ArgumentConfig {
		return &graphql.ArgumentConfig{
			Description:  p.Description,
			Type:         p.Type,
			DefaultValue: p.Default,
		}
	})
}
