package cryptodev

import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// VDevConfig configures a virtual crypto device.
type VDevConfig struct {
	Config
	// Socket is the preferred NUMA socket.
	Socket eal.NumaSocket
	// Drivers contains preferred drivers. Default is openssl.
	Drivers []string
}

func (cfg *VDevConfig) applyDefaults() {
	cfg.Config.applyDefaults()
	if len(cfg.Drivers) == 0 {
		cfg.Drivers = []string{"openssl"}
	}
}

// CreateVDev creates a virtual crypto device.
func CreateVDev(cfg VDevConfig) (cd *CryptoDev, e error) {
	cfg.applyDefaults()
	args := map[string]any{
		"max_nb_queue_pairs": cfg.NQueuePairs,
	}
	if !cfg.Socket.IsAny() {
		args["socket_id"] = cfg.Socket.ID()
	}

	var vdev *eal.VDev
	drvErrors := []error{}
	for _, drv := range cfg.Drivers {
		name := fmt.Sprintf("crypto_%s_%s", drv, eal.AllocObjectID("cryptodev.Driver["+drv+"]"))
		vdev, e = eal.NewVDev(name, args, cfg.Socket)
		if e == nil {
			break
		}
		drvErrors = append(drvErrors, fmt.Errorf("cryptodev[%s] %w", drv, e))
	}
	if vdev == nil {
		return nil, errors.Join(drvErrors...)
	}

	if cd, e = New(vdev, cfg.Config); e != nil {
		return nil, e
	}
	return cd, nil
}
