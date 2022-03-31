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
		Type:        gqlserver.NewNonNullList(ealthread.GqlWorkerType),
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
		Type:        graphql.NewNonNull(iface.GqlFaceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(withCommonFields).Face(), nil
		},
	}

	return fields
}

// NewNodeType creates a NodeType for traffic generator element.
func NewNodeType(name string, value withCommonFields, retrieve *func(iface.ID) any) (nt *gqlserver.NodeType) {
	nt = gqlserver.NewNodeTypeNamed(name, value)
	nt.GetID = func(source any) string {
		return strconv.Itoa(int(source.(withCommonFields).Face().ID()))
	}
	nt.Retrieve = func(id string) (any, error) {
		i, e := strconv.Atoi(id)
		if e != nil || *retrieve == nil {
			return nil, nil
		}
		return (*retrieve)(iface.ID(i)), nil
	}
	return nt
}
