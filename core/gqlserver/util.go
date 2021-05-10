package gqlserver

import (
	"go/ast"
	"reflect"
	"strings"

	go2gql_scalars "github.com/EGT-Ukraine/go2gql/api/scalars"
	tools_scalars "github.com/bhoriuchi/graphql-go-tools/scalars"
	"github.com/graphql-go/graphql"
	"go.uber.org/zap"
)

// Scalar types.
var (
	JSON           = tools_scalars.ScalarJSON
	NonNullJSON    = graphql.NewNonNull(JSON)
	Bytes          = go2gql_scalars.GraphQLBytesScalar
	NonNullID      = graphql.NewNonNull(graphql.ID)
	NonNullBoolean = graphql.NewNonNull(graphql.Boolean)
	NonNullInt     = graphql.NewNonNull(graphql.Int)
	NonNullString  = graphql.NewNonNull(graphql.String)
)

// NewNonNullList constructs a non-null list type.
// NewNonNullList(T) returns [T!]!.
// NewNonNullList(T, true) returns [T]!.
func NewNonNullList(ofType graphql.Type, optionalNullable ...bool) graphql.Type {
	if len(optionalNullable) < 1 || !optionalNullable[0] {
		if _, ok := ofType.(*graphql.NonNull); !ok {
			ofType = graphql.NewNonNull(ofType)
		}
	}
	return graphql.NewNonNull(graphql.NewList(ofType))
}

// Optional turns invalid value to nil.
//  Optional(value) considers the value invalid if it is zero.
//  Optional(value, valid) considers the value invalid if valid is false.
func Optional(value interface{}, optionalValid ...bool) interface{} {
	ok := true
	switch len(optionalValid) {
	case 0:
		ok = !reflect.ValueOf(value).IsZero()
	case 1:
		ok = optionalValid[0]
	default:
		panic("Optional: bad arguments")
	}

	if ok {
		return value
	}
	return nil
}

// MethodResolver creates a FieldResolveFn that invokes the named method with p.Source receiver and no arguments.
func MethodResolver(methodName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		val := reflect.ValueOf(p.Source)
		method := val.MethodByName(methodName)
		result := method.Call(nil)
		return result[0].Interface(), nil
	}
}

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

func (m FieldTypes) bindFields(typ reflect.Type, structPath []string,
	saveField func(name string, t graphql.Type, resolve graphql.FieldResolveFn)) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		logger.Panic("BindFields only accepts struct type", zap.Stringer("type", typ))
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !ast.IsExported(field.Name) {
			continue
		}

		fieldPath := append([]string{}, structPath...)
		fieldPath = append(fieldPath, field.Name)

		if field.Anonymous {
			m.bindFields(field.Type, fieldPath, saveField)
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

		saveField(name, ft, func(p graphql.ResolveParams) (interface{}, error) {
			obj := reflect.ValueOf(p.Source)
			for _, fieldName := range fieldPath {
				if obj.Kind() == reflect.Ptr {
					if obj.IsZero() {
						return nil, nil
					}
					obj = obj.Elem()
				}

				obj = obj.FieldByName(fieldName)
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
	typ := reflect.TypeOf(value)
	fields := make(graphql.Fields)
	m.bindFields(typ, nil, func(name string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.Field{
			Type:    t,
			Resolve: resolve,
		}
	})
	return fields
}

// BindInputFields creates graphql.InputObjectConfigFieldMap from a struct.
func BindInputFields(value interface{}, m FieldTypes) graphql.InputObjectConfigFieldMap {
	typ := reflect.TypeOf(value)
	fields := make(graphql.InputObjectConfigFieldMap)
	m.bindFields(typ, nil, func(name string, t graphql.Type, resolve graphql.FieldResolveFn) {
		fields[name] = &graphql.InputObjectFieldConfig{Type: t}
	})
	return fields
}
