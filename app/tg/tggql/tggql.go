// Package tggql contains shared functions among traffic generator elements.
package tggql

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

type withCommonFields interface {
	Workers() []ealthread.ThreadWithRole
	Face() iface.Face
}

// CommonFields adds 'workers' and 'face' fields.
func CommonFields(fields graphql.Fields) graphql.Fields {
	if fields == nil {
		fields = graphql.Fields{}
	}

	fields["workers"] = &graphql.Field{
		Description: "Worker threads.",
		Type:        gqlserver.NewNonNullList(ealthread.GqlWorkerType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			lcores := []eal.LCore{}
			for _, w := range p.Source.(withCommonFields).Workers() {
				lcores = append(lcores, w.LCore())
			}
			return lcores, nil
		},
	}

	fields["face"] = &graphql.Field{
		Description: "Face on which this traffic generator operates.",
		Type:        graphql.NewNonNull(iface.GqlFaceType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(withCommonFields).Face(), nil
		},
	}

	return fields
}

// NodeConfig constructs NodeConfig for traffic generator element.
func NodeConfig[T withCommonFields](retrieve *func(iface.ID) T) (nc gqlserver.NodeConfig[T]) {
	nc.GetID = func(source T) string {
		return strconv.Itoa(int(source.Face().ID()))
	}
	nc.RetrieveInt = func(id int) T {
		var zero T
		if *retrieve == nil {
			return zero
		}
		return (*retrieve)(iface.ID(id))
	}
	return
}
