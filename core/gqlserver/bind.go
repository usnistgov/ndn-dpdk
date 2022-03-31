package gqlserver

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"go.uber.org/zap"
)

// fieldIndexResolver provides a graphql.FieldResolveFn that extracts a nested field corresponding to index.
type fieldIndexResolver []int

func (index fieldIndexResolver) Resolve(p graphql.ResolveParams) (any, error) {
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

// Merge combines two or more FieldTypes maps to a new FieldTypes map.
func (m FieldTypes) Merge(a ...FieldTypes) (s FieldTypes) {
	s = FieldTypes{}
	for _, v := range append([]FieldTypes{m}, a...) {
		for k, t := range v {
			s[k] = t
		}
	}
	return s
}

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

func (m FieldTypes) bindFields(value any, saveField func(p fieldInfo)) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for _, field := range reflect.VisibleFields(typ) {
		if !field.IsExported() {
			continue
		}

		jsonTag, ok := field.Tag.Lookup("json")
		if !ok {
			continue
		}
		jsonTokens := strings.Split(jsonTag, ",")

		p := fieldInfo{}
		p.Name = jsonTokens[0]
		if p.Name == "-" {
			continue
		}

		p.Description = field.Tag.Get("gqldesc")

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

		index := append(fieldIndexResolver{}, field.Index...)
		p.Resolve = index.Resolve
		saveField(p)
	}
}

type fieldInfo struct {
	Name        string
	Description string
	Default     any

	Type    graphql.Type
	Resolve graphql.FieldResolveFn
}

// BindFields creates graphql.Field from a struct.
func BindFields(value any, m FieldTypes) graphql.Fields {
	fields := graphql.Fields{}
	m.bindFields(value, func(p fieldInfo) {
		fields[p.Name] = &graphql.Field{
			Description: p.Description,
			Type:        p.Type,
			Resolve:     p.Resolve,
		}
	})
	return fields
}

// BindInputFields creates graphql.InputObjectConfigFieldMap from a struct.
func BindInputFields(value any, m FieldTypes) graphql.InputObjectConfigFieldMap {
	fields := graphql.InputObjectConfigFieldMap{}
	m.bindFields(value, func(p fieldInfo) {
		fields[p.Name] = &graphql.InputObjectFieldConfig{
			Description:  p.Description,
			Type:         p.Type,
			DefaultValue: p.Default,
		}
	})
	return fields
}

// BindArguments creates graphql.FieldConfigArgument from a struct.
func BindArguments(value any, m FieldTypes) graphql.FieldConfigArgument {
	fields := graphql.FieldConfigArgument{}
	m.bindFields(value, func(p fieldInfo) {
		fields[p.Name] = &graphql.ArgumentConfig{
			Description:  p.Description,
			Type:         p.Type,
			DefaultValue: p.Default,
		}
	})
	return fields
}
