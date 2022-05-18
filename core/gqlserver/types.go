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

// NewListNonNullList constructs [T]! type.
func NewListNonNullList(ofType graphql.Type) graphql.Type {
	return graphql.NewNonNull(graphql.NewList(ofType))
}

// NewListNonNullElem constructs [T!] type.
func NewListNonNullElem(ofType graphql.Type) graphql.Type {
	return graphql.NewList(toNonNull(ofType))
}

// NewListNonNullBoth constructs [T!]! type.
func NewListNonNullBoth(ofType graphql.Type) graphql.Type {
	return graphql.NewNonNull(graphql.NewList(toNonNull(ofType)))
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

// Optional turns zero value to nil.
func Optional(value any) any {
	if reflect.ValueOf(value).IsZero() {
		return nil
	}
	return value
}
