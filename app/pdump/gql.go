package pdump

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsingleton"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlLCore is the LCore used for writer created via GraphQL.
	GqlLCore  eal.LCore
	gqlWriter gqlsingleton.Singleton
)

// GraphQL types.
var (
	GqlDirectionEnum         *graphql.Enum
	GqlEthGrabEnum           *graphql.Enum
	GqlNameFilterEntryInput  *graphql.InputObject
	GqlNameFilterEntryType   *graphql.Object
	GqlWriterNodeType        *gqlserver.NodeType
	GqlWriterType            *graphql.Object
	GqlFaceSourceNodeType    *gqlserver.NodeType
	GqlFaceSourceType        *graphql.Object
	GqlEthPortSourceNodeType *gqlserver.NodeType
	GqlEthPortSourceType     *graphql.Object
	GqlSourceType            *graphql.Union
)

func init() {
	GqlDirectionEnum = gqlserver.NewStringEnum("PdumpDirection", "Packet dump traffic direction.", DirIncoming, DirOutgoing)
	GqlEthGrabEnum = gqlserver.NewStringEnum("PdumpEthGrab", "Packet dump Ethernet port grab position.", EthGrabRxUnmatched)
	GqlNameFilterEntryInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "PdumpNameFilterEntryInput",
		Description: "Packet dump name filter entry.",
		Fields: gqlserver.BindInputFields(NameFilterEntry{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}): ndni.GqlNameType,
		}),
	})
	GqlNameFilterEntryType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "PdumpNameFilterEntry",
		Description: "Packet dump name filter entry.",
		Fields: gqlserver.BindFields(NameFilterEntry{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}): ndni.GqlNameType,
		}),
	})

	GqlWriterNodeType = gqlserver.NewNodeType((*Writer)(nil))
	gqlWriter.SetNodeType(GqlWriterNodeType)

	GqlWriterType = graphql.NewObject(GqlWriterNodeType.Annotate(graphql.ObjectConfig{
		Name:        "PdumpWriter",
		Description: "Packet dump writer.",
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
	}))
	GqlWriterNodeType.Register(GqlWriterType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createPdumpWriter",
		Description: "Start packet dump writer.",
		Args: graphql.FieldConfigArgument{
			"filename": &graphql.ArgumentConfig{
				Description: "Output file name.",
				Type:        gqlserver.NonNullString,
			},
			"maxSize": &graphql.ArgumentConfig{
				Description: "Maximum output file size in bytes. Storage will be pre-allocated.",
				Type:        graphql.Int,
			},
		},
		Type: graphql.NewNonNull(GqlWriterType),
		Resolve: gqlWriter.CreateWith(func(p graphql.ResolveParams) (any, error) {
			if !GqlLCore.Valid() || GqlLCore.IsBusy() {
				return nil, fmt.Errorf("no LCore for %s role; check activation parameters and ensure there's no other writer running", Role)
			}

			cfg := WriterConfig{
				Filename: p.Args["filename"].(string),
			}
			if maxSize, ok := p.Args["maxSize"]; ok {
				cfg.MaxSize = maxSize.(int)
			}
			w, e := NewWriter(cfg)
			if e != nil {
				return nil, e
			}
			w.SetLCore(GqlLCore)
			ealthread.Launch(w)
			return w, nil
		}),
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "pdumpWriters",
		Description: "List of active packet dump writers.",
		Type:        gqlserver.NewNonNullList(GqlWriterType),
		Resolve:     gqlWriter.QueryList,
	})

	GqlFaceSourceNodeType = gqlserver.NewNodeType((*FaceSource)(nil))
	GqlFaceSourceNodeType.GetID = func(source any) string {
		s := source.(*FaceSource)
		return s.key.String()
	}
	GqlFaceSourceNodeType.Retrieve = func(id string) (any, error) {
		fd, e := parseFaceDir(id)
		if e != nil {
			return nil, e
		}

		sourcesMutex.Lock()
		defer sourcesMutex.Unlock()
		return faceSources[fd], nil
	}
	GqlFaceSourceNodeType.Delete = func(source any) error {
		fs := source.(*FaceSource)
		return fs.Close()
	}

	GqlFaceSourceType = graphql.NewObject(GqlFaceSourceNodeType.Annotate(graphql.ObjectConfig{
		Name:        "PdumpFaceSource",
		Description: "Packet dump source attached to a face on a single direction.",
		Fields: graphql.Fields{
			"writer": &graphql.Field{
				Description: "Destination writer.",
				Type:        graphql.NewNonNull(GqlWriterType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*FaceSource)
					return s.Writer, nil
				},
			},
			"face": &graphql.Field{
				Description: "Source face.",
				Type:        graphql.NewNonNull(iface.GqlFaceType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*FaceSource)
					return s.Face, nil
				},
			},
			"dir": &graphql.Field{
				Description: "Traffic direction.",
				Type:        graphql.NewNonNull(GqlDirectionEnum),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*FaceSource)
					return s.Dir, nil
				},
			},
			"names": &graphql.Field{
				Description: "Name filter.",
				Type:        gqlserver.NewNonNullList(GqlNameFilterEntryType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*FaceSource)
					return s.Names, nil
				},
			},
		},
	}))
	GqlFaceSourceNodeType.Register(GqlFaceSourceType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createPdumpFaceSource",
		Description: "Create packet dump source attached to a face on a single direction.",
		Args: graphql.FieldConfigArgument{
			"writer": &graphql.ArgumentConfig{
				Description: "Destination writer.",
				Type:        gqlserver.NonNullID,
			},
			"face": &graphql.ArgumentConfig{
				Description: "Source face.",
				Type:        gqlserver.NonNullID,
			},
			"dir": &graphql.ArgumentConfig{
				Description: "Traffic direction.",
				Type:        graphql.NewNonNull(GqlDirectionEnum),
			},
			"names": &graphql.ArgumentConfig{
				Description: "Name filter.",
				Type:        gqlserver.NewNonNullList(GqlNameFilterEntryInput),
			},
		},
		Type: graphql.NewNonNull(GqlFaceSourceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			cfg := FaceConfig{}
			gqlserver.RetrieveNodeOfType(GqlWriterNodeType, p.Args["writer"], &cfg.Writer)
			gqlserver.RetrieveNodeOfType(iface.GqlFaceNodeType, p.Args["face"], &cfg.Face)
			cfg.Dir = p.Args["dir"].(Direction)
			jsonhelper.Roundtrip(p.Args["names"], &cfg.Names)
			return NewFaceSource(cfg)
		},
	})

	GqlEthPortSourceNodeType = gqlserver.NewNodeType((*EthPortSource)(nil))
	GqlEthPortSourceNodeType.GetID = func(source any) string {
		s := source.(*EthPortSource)
		return strconv.Itoa(s.Port.EthDev().ID())
	}
	GqlEthPortSourceNodeType.Retrieve = func(id string) (any, error) {
		ethDevObj, _ := ethdev.GqlEthDevNodeType.Retrieve(id)
		ethDev, _ := ethDevObj.(ethdev.EthDev)
		port := ethport.Find(ethDev)
		if port == nil {
			return nil, nil
		}

		sourcesMutex.Lock()
		defer sourcesMutex.Unlock()
		return ethPortSources[port], nil
	}
	GqlEthPortSourceNodeType.Delete = func(source any) error {
		s := source.(*EthPortSource)
		return s.Close()
	}

	GqlEthPortSourceType = graphql.NewObject(GqlEthPortSourceNodeType.Annotate(graphql.ObjectConfig{
		Name:        "PdumpEthPortSource",
		Description: "Packet dump source attached to a face on a single direction.",
		Fields: graphql.Fields{
			"writer": &graphql.Field{
				Description: "Destination writer.",
				Type:        graphql.NewNonNull(GqlWriterType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*EthPortSource)
					return s.Writer, nil
				},
			},
			"port": &graphql.Field{
				Description: "Ethernet device.",
				Type:        graphql.NewNonNull(ethdev.GqlEthDevType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*EthPortSource)
					return s.Port.EthDev(), nil
				},
			},
			"grab": &graphql.Field{
				Description: "Grab opportunity.",
				Type:        graphql.NewNonNull(GqlEthGrabEnum),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					s := p.Source.(*EthPortSource)
					return s.Grab, nil
				},
			},
		},
	}))
	GqlEthPortSourceNodeType.Register(GqlEthPortSourceType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createPdumpEthPortSource",
		Description: "Create packet dump source attached to an Ethernet port on a grab opportunity.",
		Args: graphql.FieldConfigArgument{
			"writer": &graphql.ArgumentConfig{
				Description: "Destination writer.",
				Type:        gqlserver.NonNullID,
			},
			"port": &graphql.ArgumentConfig{
				Description: "Ethernet device.",
				Type:        gqlserver.NonNullID,
			},
			"grab": &graphql.ArgumentConfig{
				Description: "Grab opportunity.",
				Type:        graphql.NewNonNull(GqlEthGrabEnum),
			},
		},
		Type: graphql.NewNonNull(GqlEthPortSourceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			cfg := EthPortConfig{}
			gqlserver.RetrieveNodeOfType(GqlWriterNodeType, p.Args["writer"], &cfg.Writer)
			var ethDev ethdev.EthDev
			if e := gqlserver.RetrieveNodeOfType(ethdev.GqlEthDevNodeType, p.Args["port"], &ethDev); e == nil {
				cfg.Port = ethport.Find(ethDev)
			}
			cfg.Grab = p.Args["grab"].(EthGrab)
			return NewEthPortSource(cfg)
		},
	})

	GqlSourceType = graphql.NewUnion(graphql.UnionConfig{
		Name:  "PdumpSource",
		Types: []*graphql.Object{GqlFaceSourceType, GqlEthPortSourceType},
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			switch p.Value.(type) {
			case *FaceSource:
				return GqlFaceSourceType
			case *EthPortSource:
				return GqlEthPortSourceType
			}
			return nil
		},
	})

	GqlWriterType.AddFieldConfig("sources", &graphql.Field{
		Description: "Packet dump sources.",
		Type:        gqlserver.NewNonNullList(GqlSourceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			sources := []any{}

			sourcesMutex.Lock()
			defer sourcesMutex.Unlock()
			for _, s := range faceSources {
				sources = append(sources, s)
			}
			for _, s := range ethPortSources {
				sources = append(sources, s)
			}

			return sources, nil
		},
	})
}
