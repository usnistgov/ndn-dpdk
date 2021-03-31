package cryptodev

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/multierr"
)

// DriverPref is a priority list of CryptoDev drivers.
type DriverPref []string

var (
	// SingleSegDrv lists CryptoDev drivers capable of computing SHA256 on single-segment mbufs.
	SingleSegDrv = DriverPref{"aesni_mb", "openssl"}

	// MultiSegDrv lists CryptoDev drivers capable of computing SHA256 on multi-segment mbufs.
	MultiSegDrv = DriverPref{"openssl"}
)

// Create constructs a CryptoDev from a list of drivers.
func (drvs DriverPref) Create(cfg Config, socket eal.NumaSocket) (cd *CryptoDev, e error) {
	cfg.applyDefaults()
	args := map[string]interface{}{
		"max_nb_queue_pairs": cfg.NQueuePairs,
	}
	if !socket.IsAny() {
		args["socket_id"] = socket.ID()
	}

	var vdev *eal.VDev
	drvErrors := []error{}
	for _, drv := range drvs {
		name := fmt.Sprintf("crypto_%s_%s", drv, eal.AllocObjectID("cryptodev.Driver["+drv+"]"))
		vdev, e = eal.NewVDev(name, args, socket)
		if e == nil {
			break
		}
		drvErrors = append(drvErrors, fmt.Errorf("cryptodev[%s] %w", drv, e))
	}
	if vdev == nil {
		return nil, multierr.Combine(drvErrors...)
	}

	if cd, e = New(vdev, cfg); e != nil {
		return nil, e
	}
	return cd, nil
}
