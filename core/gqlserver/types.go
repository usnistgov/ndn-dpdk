package gqlserver

import (
	"reflect"

	go2gql_scalars "github.com/EGT-Ukraine/go2gql/api/scalars"
	tools_scalars "github.com/bhoriuchi/graphql-go-tools/scalars"
	"github.com/graphql-go/graphql"
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

func toNonNull(ofType graphql.Type) graphql.Type {
	if _, ok := ofType.(*graphql.NonNull); ok {
		return ofType
	}
	return graphql.NewNonNull(ofType)
}

// NewNonNullList constructs a non-null list type.
// NewNonNullList(T) returns [T!]!.
// NewNonNullList(T, true) returns [T]!.
func NewNonNullList(ofType graphql.Type, optionalNullable ...bool) graphql.Type {
	if len(optionalNullable) < 1 || !optionalNullable[0] {
		ofType = toNonNull(ofType)
	}
	return graphql.NewNonNull(graphql.NewList(ofType))
}

// NewStringEnum constructs an enum type.
// The underlying type of value must be string.
func NewStringEnum(name, desc string, values ...interface{}) *graphql.Enum {
	vm := graphql.EnumValueConfigMap{}
	for _, value := range values {
		val := reflect.ValueOf(value)
		vm[val.String()] = &graphql.EnumValueConfig{Value: value}
	}
	return graphql.NewEnum(graphql.EnumConfig{
		Name:        name,
		Description: desc,
		Values:      vm,
	})
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
