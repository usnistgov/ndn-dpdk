package ndni

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// GraphQL types.
var (
	GqlNameType              *graphql.Scalar
	GqlInterestTemplateInput *graphql.InputObject
	GqlDataGenInput          *graphql.InputObject
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

	GqlInterestTemplateInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "InterestTemplateInput",
		Description: "Interest template.",
		Fields: gqlserver.BindInputFields(InterestTemplateConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}):                 gqlserver.NonNullString,
			reflect.TypeOf(nnduration.Milliseconds(0)): nnduration.GqlMilliseconds,
		}),
	})

	GqlDataGenInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "DataGenInput",
		Description: "Data generator template.",
		Fields: gqlserver.BindInputFields(DataGenConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}):                 gqlserver.NonNullString,
			reflect.TypeOf(nnduration.Milliseconds(0)): nnduration.GqlMilliseconds,
		}),
	})
}
