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
	GqlNameType                   *graphql.Scalar
	GqlInterestTemplateInput      *graphql.InputObject
	GqlInterestTemplateFieldTypes gqlserver.FieldTypes
	GqlDataGenInput               *graphql.InputObject
)

func init() {
	GqlNameType = graphql.NewScalar(graphql.ScalarConfig{
		Name:        "Name",
		Description: "The `Name` scalar type represents an NDN name.",
		Serialize: func(value any) any {
			switch v := value.(type) {
			case ndn.Name:
				return v.String()
			case *ndn.Name:
				return v.String()
			}
			return nil
		},
		ParseValue: func(value any) any {
			switch v := value.(type) {
			case string:
				return ndn.ParseName(v)
			case *string:
				return ndn.ParseName(*v)
			}
			return nil
		},
		ParseLiteral: func(value ast.Value) any {
			switch v := value.(type) {
			case *ast.StringValue:
				return ndn.ParseName(v.Value)
			}
			return nil
		},
	})

	GqlInterestTemplateFieldTypes = gqlserver.FieldTypes{
		reflect.TypeFor[ndn.Name]():                gqlserver.NonNullString,
		reflect.TypeFor[nnduration.Milliseconds](): nnduration.GqlMilliseconds,
	}
	GqlInterestTemplateInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "InterestTemplateInput",
		Description: "Interest template.",
		Fields:      gqlserver.BindInputFields[InterestTemplateConfig](GqlInterestTemplateFieldTypes),
	})

	GqlDataGenInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "DataGenInput",
		Description: "Data generator template.",
		Fields: gqlserver.BindInputFields[DataGenConfig](gqlserver.FieldTypes{
			reflect.TypeFor[ndn.Name]():                gqlserver.NonNullString,
			reflect.TypeFor[nnduration.Milliseconds](): nnduration.GqlMilliseconds,
		}),
	})
}
