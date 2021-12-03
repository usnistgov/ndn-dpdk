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
		reflect.TypeOf(DriverKind("")):           GqlDriverKindEnum,
		reflect.TypeOf(pciaddr.PCIAddress{}):     graphql.String,
		reflect.TypeOf(map[string]interface{}{}): gqlserver.JSON,
	}
}
