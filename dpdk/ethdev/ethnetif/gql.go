package ethnetif

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
)

// GraphQL types.
var (
	GqlDriverKindEnum   *graphql.Enum
	GqlConfigFieldTypes gqlserver.FieldTypes
)

func init() {
	GqlDriverKindEnum = gqlserver.NewStringEnum("NetifDriverKind", "", DriverPCI, DriverXDP, DriverAfPacket)

	GqlConfigFieldTypes = gqlserver.FieldTypes{
		reflect.TypeFor[DriverKind]():         GqlDriverKindEnum,
		reflect.TypeFor[pciaddr.PCIAddress](): graphql.String,
		reflect.TypeFor[map[string]any]():     gqlserver.JSON,
	}
}
