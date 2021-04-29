package tggql

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var (
	// GqlTrafficGen is the *tg.TrafficGen instance accessible via GraphQL.
	GqlTrafficGen interface{}

	errNoGqlTrafficGen = errors.New("TrafficGen unavailable")
)

type withCommonFields interface {
	Workers() []ealthread.ThreadWithRole
	Face() iface.Face
}

// CommonFields adds 'workers' and 'face' fields.
func CommonFields(fields graphql.Fields) graphql.Fields {
	fields["workers"] = &graphql.Field{
		Description: "Worker threads.",
		Type:        gqlserver.NewNonNullList(ealthread.GqlWorkerType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Source.(withCommonFields).Workers(), nil
		},
	}

	fields["face"] = &graphql.Field{
		Description: "Face used by traffic generator.",
		Type:        graphql.NewNonNull(iface.GqlFaceType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Source.(withCommonFields).Face(), nil
		},
	}

	return fields
}

// Get returns GqlTrafficGen.Task(id)[taskField].
func Get(id iface.ID, taskField string) interface{} {
	if GqlTrafficGen == nil {
		return nil
	}
	gen := reflect.ValueOf(GqlTrafficGen)
	task := gen.MethodByName("Task").Call([]reflect.Value{reflect.ValueOf(id)})[0]
	if task.IsNil() {
		return nil
	}
	return task.Elem().FieldByName(taskField).Interface()
}

// NewNodeType creates a NodeType for traffic generator element.
func NewNodeType(value withCommonFields, taskField string) (nt *gqlserver.NodeType) {
	nt = gqlserver.NewNodeType(value)
	nt.GetID = func(source interface{}) string {
		return strconv.Itoa(int(source.(withCommonFields).Face().ID()))
	}
	nt.Retrieve = func(id string) (interface{}, error) {
		if GqlTrafficGen == nil {
			return nil, errNoGqlTrafficGen
		}
		i, e := strconv.Atoi(id)
		if e != nil {
			return nil, nil
		}
		return Get(iface.ID(i), taskField), nil
	}

	return nt
}

// AddFaceField adds a field to iface.GqlFaceType.
func AddFaceField(name, description, taskField string, gqlType graphql.Output) {
	iface.GqlFaceType.AddFieldConfig(name, &graphql.Field{
		Description: description,
		Type:        gqlType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlTrafficGen == nil {
				return nil, nil
			}
			face := p.Source.(iface.Face)
			return Get(face.ID(), taskField), nil
		},
	})
}
