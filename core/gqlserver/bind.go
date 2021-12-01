package gqlserver

import (
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"go.uber.org/zap"
)

type fieldTag struct {
	Skip        bool
	Name        string
	OmitEmpty   bool
	Description string
}

func parseFieldTag(field reflect.StructField) (tag fieldTag) {
	jsonTag, ok := field.Tag.Lookup("json")
	if !ok {
		tag.Skip = true
		return
	}

	jsonTokens := strings.Split(jsonTag, ",")
	tag.Name = jsonTokens[0]
	tag.Skip = tag.Name == "-"
	tag.OmitEmpty = len(jsonTokens) >= 2 && jsonTokens[1] == "omitempty"

	tag.Description = field.Tag.Get("gqldesc")
	return
}

// fieldIndexResolver provides a graphql.FieldResolveFn that extracts a nested field corresponding to index.
type fieldIndexResolver []int

func (index fieldIndexResolver) Resolve(p graphql.ResolveParams) (interface{}, error) {
	v := reflect.ValueOf(p.Source)
	// unlike v.FieldByIndex, this logic does not panic upon nil pointer
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil, nil
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v.Interface(), nil
}

// FieldTypes contains known GraphQL types of fields.
type FieldTypes map[reflect.Type]graphql.Type

func (m FieldTypes) resolveType(typ reflect.Type) graphql.Type {
	if t := m[typ]; t != nil {
		if kind := typ.Kind(); kind == reflect.Ptr || kind == reflect.Slice {
			return t
		}
		return toNonNull(t)
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
		// NaN is null, so this would not allow NaN
		return graphql.NewNonNull(graphql.Float)
	case reflect.Uint64, reflect.Int64, reflect.String:
		return NonNullString
	}

	logger.Panic("FieldTypes cannot resolve type", zap.Stringer("type", typ))
	return nil
}

func (m FieldTypes) bindFields(value interface{}, saveField func(name, desc string, t graphql.Type, resolve graphql.FieldResolveFn)) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for _, field := range reflect.VisibleFields(typ) {
		tag := parseFieldTag(field)
		if !field.IsExported() || tag.Skip {
			continue
		}

		t := m.resolveType(field.Type)
		if tag.OmitEmpty {
			t = graphql.GetNullable(t).(graphql.Type)
		}

		index := append(fieldIndexResolver{}, field.Index...)
		saveField(tag.Name, tag.Description, t, index.Resolve)
	}
}

// BindFields creates graphql.Field from a struct.
func BindFields(value interface{}, m FieldTypes) graphql.Fields {
	fields := graphql.Fields{}
	m.bindFields(value, func(name, desc string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.Field{
			Type:        t,
			Description: desc,
			Resolve:     resolve,
		}
	})
	return fields
}

// BindInputFields creates graphql.InputObjectConfigFieldMap from a struct.
func BindInputFields(value interface{}, m FieldTypes) graphql.InputObjectConfigFieldMap {
	fields := graphql.InputObjectConfigFieldMap{}
	m.bindFields(value, func(name, desc string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.InputObjectFieldConfig{
			Type:        t,
			Description: desc,
		}
	})
	return fields
}
