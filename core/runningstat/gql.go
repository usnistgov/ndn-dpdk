package runningstat

import (
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlSnapshotType *graphql.Object
)

func init() {
	GqlSnapshotType = graphql.NewObject(graphql.ObjectConfig{
		Name: "RunningStatSnapshot",
		Fields: graphql.Fields{
			"count": &graphql.Field{
				Type: gqlserver.NonNullUint64,
			},
			"len": &graphql.Field{
				Type: gqlserver.NonNullUint64,
			},
			"min": &graphql.Field{
				Type: graphql.Float,
			},
			"max": &graphql.Field{
				Type: graphql.Float,
			},
			"mean": &graphql.Field{
				Type: graphql.Float,
			},
			"variance": &graphql.Field{
				Type: graphql.Float,
			},
			"stdev": &graphql.Field{
				Type: graphql.Float,
			},
			"m1": &graphql.Field{
				Type: graphql.Float,
			},
			"m2": &graphql.Field{
				Type: graphql.Float,
			},
		},
	})
}

var _ graphql.FieldResolver = Snapshot{}

// Resolve implements graphql.FieldResolver interface.
func (s Snapshot) Resolve(p graphql.ResolveParams) (interface{}, error) {
	val := reflect.ValueOf(s)
	typ := val.Type()
	for i, nMethods := 0, val.NumMethod(); i < nMethods; i++ {
		if strings.EqualFold(typ.Method(i).Name, p.Info.FieldName) {
			result := val.Method(i).Call(nil)
			return result[0].Interface(), nil
		}
	}
	panic("field not found")
}
