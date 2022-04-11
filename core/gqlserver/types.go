package gqlserver

import (
	"reflect"

	go2gql_scalars "github.com/EGT-Ukraine/go2gql/api/scalars"
	tools_scalars "github.com/bhoriuchi/graphql-go-tools/scalars"
	"github.com/graphql-go/graphql"
)

// Scalar types.
var (
	JSON   = tools_scalars.ScalarJSON
	Bytes  = go2gql_scalars.GraphQLBytesScalar
	Uint64 = go2gql_scalars.GraphQLUInt64Scalar
	Int64  = go2gql_scalars.GraphQLInt64Scalar

	NonNullJSON    = graphql.NewNonNull(JSON)
	NonNullUint64  = graphql.NewNonNull(Uint64)
	NonNullInt64   = graphql.NewNonNull(Int64)
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
	nullable := false
	switch len(optionalNullable) {
	case 0:
	case 1:
		nullable = optionalNullable[0]
	default:
		panic("NewNonNullList: bad arguments")
	}

	if !nullable {
		ofType = toNonNull(ofType)
	}
	return graphql.NewNonNull(graphql.NewList(ofType))
}

// NewStringEnum constructs an enum type.
func NewStringEnum[T ~string](name, desc string, values ...T) *graphql.Enum {
	vm := graphql.EnumValueConfigMap{}
	for _, value := range values {
		vm[string(value)] = &graphql.EnumValueConfig{Value: value}
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
func Optional(value any, optionalValid ...bool) any {
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
