package pdump

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlLCore is the LCore used for writer created via GraphQL.
	GqlLCore eal.LCore

	gqlMutex sync.Mutex

	errNoLCore = fmt.Errorf("no LCore for %s role", Role)
)

// GraphQL types.
var (
	GqlDirectionEnum        *graphql.Enum
	GqlNameFilterEntryInput *graphql.InputObject
	GqlFaceConfigInput      *graphql.InputObject
)

func init() {
	GqlDirectionEnum = graphql.NewEnum(graphql.EnumConfig{
		Name:        "PdumpDirection",
		Description: "Packet dumper traffic direction.",
		Values: graphql.EnumValueConfigMap{
			string(DirIncoming): &graphql.EnumValueConfig{Value: DirIncoming},
			string(DirOutgoing): &graphql.EnumValueConfig{Value: DirOutgoing},
		},
	})
	GqlNameFilterEntryInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "PdumpNameFilterEntryInput",
		Description: "Packet dumper name filter entry.",
		Fields: gqlserver.BindInputFields(NameFilterEntry{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}): ndni.GqlNameType,
		}),
	})
	GqlFaceConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "PdumpFaceConfigInput",
		Description: "Face packet dumper configuration.",
		Fields: gqlserver.BindInputFields(FaceConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(Direction("")):     GqlDirectionEnum,
			reflect.TypeOf(NameFilterEntry{}): GqlNameFilterEntryInput,
		}),
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "pdump",
		Description: "Dump packets from faces.",
		Args: graphql.FieldConfigArgument{
			"filename": &graphql.ArgumentConfig{
				Description: "Output file name.",
				Type:        gqlserver.NonNullString,
			},
			"maxSize": &graphql.ArgumentConfig{
				Description: "Maximum output file size in bytes.",
				Type:        graphql.Int,
			},
			"duration": &graphql.ArgumentConfig{
				Description: "Collection duration.",
				Type:        graphql.NewNonNull(nnduration.GqlNanoseconds),
			},
			"faces": &graphql.ArgumentConfig{
				Description: "Faces to collect from.",
				Type:        gqlserver.NewNonNullList(GqlFaceConfigInput),
			},
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			gqlMutex.Lock()
			defer gqlMutex.Unlock()
			if !GqlLCore.Valid() || GqlLCore.IsBusy() {
				return nil, errNoLCore
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

			faceDumps := []*Face{}
			defer func() {
				for _, pd := range faceDumps {
					pd.Close()
				}
				w.Close()
			}()

			var faceConfigs []FaceConfig
			jsonhelper.Roundtrip(p.Args["faces"], &faceConfigs)
			for i, f := range faceConfigs {
				var face iface.Face
				if e := gqlserver.RetrieveNodeOfType(iface.GqlFaceNodeType, f.ID, &face); e != nil {
					return nil, fmt.Errorf("face %s at index %d not found", f.ID, i)
				}
				pd, e := DumpFace(face, w, f)
				if e != nil {
					return nil, fmt.Errorf("dumper[%d]: %w", i, e)
				}
				faceDumps = append(faceDumps, pd)
			}

			time.Sleep(p.Args["duration"].(nnduration.Nanoseconds).Duration())
			return true, nil
		},
	})
}
