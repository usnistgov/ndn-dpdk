package nnduration

import (
	"reflect"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
)

// GraphQL types.
var (
	GqlMilliseconds = makeGqlType(reflect.TypeFor[Milliseconds]())
	GqlNanoseconds  = makeGqlType(reflect.TypeFor[Nanoseconds]())
)

type durationer interface {
	Duration() time.Duration
}

func makeGqlType(typ reflect.Type) *graphql.Scalar {
	return graphql.NewScalar(graphql.ScalarConfig{
		Name:        "NN" + typ.Name(),
		Description: "Non-negative " + strings.ToLower(typ.Name()) + ", either a non-negative integer or a duration string recognized by time.ParseDuration.",
		Serialize: func(value any) any {
			return value.(durationer).Duration().String()
		},
		ParseValue: func(value any) any {
			ptr := reflect.New(typ)
			if e := jsonhelper.Roundtrip(value, ptr.Interface()); e != nil {
				return nil
			}
			return ptr.Elem().Interface()
		},
		ParseLiteral: func(valueAST ast.Value) any {
			ptr := reflect.New(typ)
			if e := jsonhelper.Roundtrip(valueAST.GetValue(), ptr.Interface()); e != nil {
				return nil
			}
			return ptr.Elem().Interface()
		},
	})
}
