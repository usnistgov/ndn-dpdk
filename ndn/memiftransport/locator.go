package memiftransport

import (
	"encoding/json"
	"errors"
	"math"
	"path"

	binutils "github.com/jfoster/binary-utilities"
	mathpkg "github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Defaults and limits.
const (
	MaxSocketNameSize = 108

	MinID = 0
	MaxID = math.MaxUint32

	MinDataroom     = 512
	MaxDataroom     = math.MaxUint16
	DefaultDataroom = 2048

	MinRingCapacity     = 1 << 1
	MaxRingCapacity     = 1 << 14
	DefaultRingCapacity = 1 << 10
)

// Role indicates memif role.
type Role string

// Role constants.
const (
	RoleServer Role = "server"
	RoleClient Role = "client"
)

// Locator identifies memif interface.
type Locator struct {
	l3.TransportQueueConfig

	// Role selects memif role.
	// Default is "client" in NDNgo library and "server" in NDN-DPDK service.
	Role Role `json:"role,omitempty"`

	// SocketName is the control socket filename.
	// It must be an absolute path, not longer than MaxSocketNameSize.
	SocketName string `json:"socketName"`

	// SocketOwner changes owner uid:gid of the socket file.
	// This is only applicable in NDN-DPDK service when creating the first memif in "server" role on a SocketName.
	SocketOwner *[2]int `json:"socketOwner,omitempty"`

	// ID is the interface identifier.
	// It must be between MinID and MaxID.
	ID int `json:"id"`

	// Dataroom is the buffer size of each packet.
	// Default is DefaultDataroom.
	// It is automatically clamped between MinDataroom and MaxDataroom.
	Dataroom int `json:"dataroom,omitempty"`

	// RingCapacity is the capacity of queue pair rings.
	// Default is DefaultRingCapacity.
	// It is automatically adjusted up to the next power of 2, and clamped between MinRingCapacity and MaxRingCapacity.
	RingCapacity int `json:"ringCapacity,omitempty"`
}

// Validate checks Locator fields.
func (loc Locator) Validate() error {
	switch loc.Role {
	case "", RoleServer, RoleClient:
	default:
		return errors.New("invalid Role")
	}
	if socketName := path.Clean(loc.SocketName); !path.IsAbs(socketName) || len(socketName) > MaxSocketNameSize {
		return errors.New("invalid SocketName")
	}
	if loc.ID < MinID || loc.ID > MaxID {
		return errors.New("invalid ID")
	}
	return nil
}

// ApplyDefaults sets empty values to defaults.
func (loc *Locator) ApplyDefaults(defaultRole Role) {
	loc.ApplyTransportQueueConfigDefaults()

	loc.SocketName = path.Clean(loc.SocketName)

	if loc.Role == "" {
		loc.Role = defaultRole
	}

	if loc.Dataroom == 0 {
		loc.Dataroom = DefaultDataroom
	} else {
		loc.Dataroom = mathpkg.MinInt(mathpkg.MaxInt(MinDataroom, loc.Dataroom), MaxDataroom)
	}

	if loc.RingCapacity == 0 {
		loc.RingCapacity = DefaultRingCapacity
	} else {
		loc.RingCapacity = mathpkg.MinInt(mathpkg.MaxInt(MinRingCapacity, loc.RingCapacity), MaxRingCapacity)
	}
	loc.RingCapacity = int(binutils.NextPowerOfTwo(int64(loc.RingCapacity)))
}

// ReverseRole returns a copy of Locator with server and client roles reversed.
func (loc Locator) ReverseRole() (reversed Locator) {
	reversed = loc
	switch loc.Role {
	case RoleServer:
		reversed.Role = RoleClient
	case RoleClient:
		reversed.Role = RoleServer
	}
	return
}

func (loc Locator) rsize() uint8 {
	return uint8(math.Log2(float64(loc.RingCapacity)))
}

// ToVDevArgs builds arguments for DPDK virtual device.
//  key: a unique key for each memif vdev; creating vdev with duplicate key would fail.
//  args: arguments passed to eal.NewVDev() function.
func (loc *Locator) ToVDevArgs() (args map[string]any, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}
	loc.ApplyDefaults(RoleServer)

	return map[string]any{
		"id":              loc.ID,
		"role":            string(loc.Role),
		"bsize":           loc.Dataroom,
		"rsize":           loc.rsize(),
		"socket":          loc.SocketName,
		"socket-abstract": "no",
		"mac":             "F2:6D:65:6D:69:66", // F2:"memif"
	}, nil
}

// ToCreateFaceLocator builds a JSON object suitable for NDN-DPDK face creation API.
func (loc *Locator) ToCreateFaceLocator() (json.RawMessage, error) {
	if e := loc.Validate(); e != nil {
		return nil, e
	}
	loc.ApplyDefaults(RoleServer)

	var m map[string]any
	if e := jsonhelper.Roundtrip(loc, &m); e != nil {
		return nil, e
	}
	m["scheme"] = "memif"

	j, e := json.Marshal(m)
	return json.RawMessage(j), e
}
