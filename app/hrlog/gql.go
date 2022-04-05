package hrlog

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsingleton"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

var (
	// GqlLCore is the LCore used for writer created via GraphQL.
	GqlLCore  eal.LCore
	gqlWriter gqlsingleton.Singleton[*Writer]
)

// GraphQL types.
var (
	GqlWriterType *gqlserver.NodeType[*Writer]
)

func init() {
	GqlWriterType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name:        "HrlogWriter",
		Description: "High resolution log writer.",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Description: "Destination filename.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					w := p.Source.(*Writer)
					return w.filename, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}, gqlWriter.NodeConfig())

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createHrlogWriter",
		Description: "Start high resolution log writer.",
		Args: graphql.FieldConfigArgument{
			"filename": &graphql.ArgumentConfig{
				Description: "Output file name.",
				Type:        gqlserver.NonNullString,
			},
			"count": &graphql.ArgumentConfig{
				Description: "Maximum number of entries. Storage will be pre-allocated.",
				Type:        graphql.Int,
			},
		},
		Type: graphql.NewNonNull(GqlWriterType.Object),
		Resolve: gqlWriter.CreateWith(func(p graphql.ResolveParams) (w *Writer, e error) {
			if !GqlLCore.Valid() || GqlLCore.IsBusy() {
				return nil, fmt.Errorf("no LCore for %s role; check activation parameters and ensure there's no other writer running", Role)
			}

			cfg := WriterConfig{
				Filename: p.Args["filename"].(string),
			}
			if count, ok := p.Args["count"]; ok {
				cfg.Count = count.(int)
			}
			w, e = NewWriter(cfg)
			if e != nil {
				return nil, e
			}
			w.SetLCore(GqlLCore)
			ealthread.Launch(w)
			return w, nil
		}),
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "hrlogWriters",
		Description: "List of active high resolution log writers.",
		Type:        gqlserver.NewNonNullList(GqlWriterType.Object),
		Resolve:     gqlWriter.QueryList,
	})
}
