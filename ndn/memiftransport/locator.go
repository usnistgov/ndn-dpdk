package memiftransport

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path"

	"github.com/FDio/vpp/extras/gomemif/memif"
	mathpkg "github.com/pkg/math"
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

// Locator identifies memif interface.
type Locator struct {
	l3.TransportQueueConfig

	// SocketName is the control socket filename.
	// It must be an absolute path, no longer than MaxSocketNameSize.
	SocketName string `json:"socketName"`

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
	if socketName := path.Clean(loc.SocketName); !path.IsAbs(socketName) || len(socketName) > MaxSocketNameSize {
		return errors.New("invalid SocketName")
	}
	if loc.ID < MinID || loc.ID > MaxID {
		return errors.New("invalid ID")
	}
	return nil
}

func (loc *Locator) applyDefaults() {
	loc.ApplyTransportQueueConfigDefaults()

	loc.SocketName = path.Clean(loc.SocketName)

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
}

func (loc Locator) rsize() uint8 {
	return uint8(math.Log2(float64(loc.RingCapacity)))
}

func (loc *Locator) toArguments(a *memif.Arguments) error {
	if e := loc.Validate(); e != nil {
		return e
	}
	loc.applyDefaults()

	a.Id = uint32(loc.ID)
	a.Name = os.Args[0]
	a.Secret = [24]byte{}
	a.MemoryConfig = memif.MemoryConfig{
		NumQueuePairs:    1,
		Log2RingSize:     loc.rsize(),
		PacketBufferSize: uint32(loc.Dataroom),
	}
	return nil
}

// ToVDevArgs builds arguments for DPDK virtual device.
func (loc *Locator) ToVDevArgs() (string, error) {
	if e := loc.Validate(); e != nil {
		return "", e
	}
	loc.applyDefaults()
	return fmt.Sprintf("id=%d,role=master,bsize=%d,rsize=%d,socket=%s,mac=%v",
		loc.ID, loc.Dataroom, loc.rsize(), loc.SocketName, AddressDPDK), nil
}

var (
	// AddressDPDK is the MAC address on DPDK side.
	AddressDPDK = net.HardwareAddr{0xF2, 0x6C, 0xE6, 0x8D, 0x9E, 0x34}

	// AddressApp is the MAC address on application side.
	AddressApp = net.HardwareAddr{0xF2, 0x71, 0x7E, 0x76, 0x5D, 0x1C}
)
