package gqlserver

import (
	"encoding/json"
	"fmt"
	"reflect"

	go2gql_scalars "github.com/EGT-Ukraine/go2gql/api/scalars"
	tools_scalars "github.com/bhoriuchi/graphql-go-tools/scalars"
	"github.com/graphql-go/graphql"
)

// Scalar types.
var (
	JSON           = tools_scalars.ScalarJSON
	Bytes          = go2gql_scalars.GraphQLBytesScalar
	NonNullID      = graphql.NewNonNull(graphql.ID)
	NonNullBoolean = graphql.NewNonNull(graphql.Boolean)
	NonNullInt     = graphql.NewNonNull(graphql.Int)
	NonNullString  = graphql.NewNonNull(graphql.String)
)

// DecodeJSON decodes JSON argument into pointer.
func DecodeJSON(arg interface{}, ptr interface{}) error {
	j, e := json.Marshal(arg)
	if e != nil {
		return fmt.Errorf("json.Marshal %w", e)
	}
	e = json.Unmarshal(j, ptr)
	if e != nil {
		return fmt.Errorf("json.Unmarshal %w", e)
	}
	return e
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
