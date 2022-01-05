package pdump

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"unsafe"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlLCore is the LCore used for writer created via GraphQL.
	GqlLCore  eal.LCore
	gqlWriter *Writer
	gqlMutex  sync.Mutex
)

// GraphQL types.
var (
	GqlDirectionEnum        *graphql.Enum
	GqlNameFilterEntryInput *graphql.InputObject
	GqlNameFilterEntryType  *graphql.Object
	GqlWriterNodeType       *gqlserver.NodeType
	GqlWriterType           *graphql.Object
	GqlFaceSourceNodeType   *gqlserver.NodeType
	GqlFaceSourceType       *graphql.Object
	GqlSourceType           *graphql.Union
)

func init() {
	GqlDirectionEnum = gqlserver.NewStringEnum("PdumpDirection", "Packet dump traffic direction.", DirIncoming, DirOutgoing)
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
	GqlWriterNodeType.GetID = func(source interface{}) string {
		w := source.(*Writer)
		return strconv.FormatUint(uint64(uintptr(unsafe.Pointer(w.c))), 16)
	}
	GqlWriterNodeType.Retrieve = func(id string) (interface{}, error) {
		gqlMutex.Lock()
		defer gqlMutex.Unlock()
		if gqlWriter != nil && GqlWriterNodeType.GetID(gqlWriter) == id {
			return gqlWriter, nil
		}
		return nil, nil
	}
	GqlWriterNodeType.Delete = func(source interface{}) error {
		w := source.(*Writer)
		if e := w.Close(); e != nil {
			return e
		}

		gqlMutex.Lock()
		defer gqlMutex.Unlock()
		gqlWriter = nil
		return nil
	}

	GqlWriterType = graphql.NewObject(GqlWriterNodeType.Annotate(graphql.ObjectConfig{
		Name:        "PdumpWriter",
		Description: "Packet dump writer.",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Description: "Destination filename.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gqlMutex.Lock()
					defer gqlMutex.Unlock()

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
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			gqlMutex.Lock()
			defer gqlMutex.Unlock()
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
			gqlWriter = w
			return w, nil
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "pdumpWriters",
		Description: "List of active packet dump writers.",
		Type:        gqlserver.NewNonNullList(GqlWriterType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			gqlMutex.Lock()
			defer gqlMutex.Unlock()

			if gqlWriter == nil {
				return []*Writer{}, nil
			}
			return []*Writer{gqlWriter}, nil
		},
	})

	GqlFaceSourceNodeType = gqlserver.NewNodeType((*FaceSource)(nil))
	GqlFaceSourceNodeType.GetID = func(source interface{}) string {
		fs := source.(*FaceSource)
		return fs.key.String()
	}
	GqlFaceSourceNodeType.Retrieve = func(id string) (interface{}, error) {
		fd, e := parseFaceDir(id)
		if e != nil {
			return nil, e
		}

		faceSourcesLock.Lock()
		defer faceSourcesLock.Unlock()
		return faceSources[fd], nil
	}
	GqlFaceSourceNodeType.Delete = func(source interface{}) error {
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
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return gqlWriter, nil
				},
			},
			"face": &graphql.Field{
				Description: "Source face.",
				Type:        graphql.NewNonNull(iface.GqlFaceType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fs := p.Source.(*FaceSource)
					return fs.Face, nil
				},
			},
			"dir": &graphql.Field{
				Description: "Traffic direction.",
				Type:        graphql.NewNonNull(GqlDirectionEnum),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fs := p.Source.(*FaceSource)
					return fs.Dir, nil
				},
			},
			"names": &graphql.Field{
				Description: "Name filter.",
				Type:        gqlserver.NewNonNullList(GqlNameFilterEntryType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fs := p.Source.(*FaceSource)
					return fs.Names, nil
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
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			cfg := FaceConfig{}
			gqlserver.RetrieveNodeOfType(GqlWriterNodeType, p.Args["writer"], &cfg.Writer)
			gqlserver.RetrieveNodeOfType(iface.GqlFaceNodeType, p.Args["face"], &cfg.Face)
			cfg.Dir = p.Args["dir"].(Direction)
			jsonhelper.Roundtrip(p.Args["names"], &cfg.Names)
			return NewFaceSource(cfg)
		},
	})

	GqlSourceType = graphql.NewUnion(graphql.UnionConfig{
		Name:  "PdumpSource",
		Types: []*graphql.Object{GqlFaceSourceType},
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			return GqlFaceSourceType
		},
	})

	GqlWriterType.AddFieldConfig("sources", &graphql.Field{
		Description: "Packet dump sources.",
		Type:        gqlserver.NewNonNullList(GqlSourceType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			sources := []interface{}{}

			faceSourcesLock.Lock()
			defer faceSourcesLock.Unlock()
			for _, fs := range faceSources {
				sources = append(sources, fs)
			}

			return sources, nil
		},
	})
}
