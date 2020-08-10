package ndni

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// GraghQL types.
var (
	GqlNameType *graphql.Scalar
)

func init() {
	GqlNameType = graphql.NewScalar(graphql.ScalarConfig{
		Name:        "Name",
		Description: "The `Name` scalar type represents an NDN name.",
		Serialize: func(value interface{}) interface{} {
			switch v := value.(type) {
			case ndn.Name:
				return v.String()
			case *ndn.Name:
				return v.String()
			}
			return nil
		},
		ParseValue: func(value interface{}) interface{} {
			switch v := value.(type) {
			case string:
				return ndn.ParseName(v)
			case *string:
				return ndn.ParseName(*v)
			}
			return nil
		},
		ParseLiteral: func(value ast.Value) interface{} {
			switch v := value.(type) {
			case *ast.StringValue:
				return ndn.ParseName(v.Value)
			}
			return nil
		},
	})
}
