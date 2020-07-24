package cryptodev

import (
	"errors"
	"fmt"
	"strings"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
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
	var args strings.Builder
	fmt.Fprintf(&args, "max_nb_queue_pairs=%d", cfg.NQueuePairs)
	if !socket.IsAny() {
		fmt.Fprintf(&args, ",socket_id=%d", socket.ID())
	}
	arg := args.String()

	var vdev *eal.VDev
	var drvErrors strings.Builder
	drvErrors.WriteString("virtual cryptodev unavailable: ")
	for _, drv := range drvs {
		name := fmt.Sprintf("crypto_%s_%s", drv, eal.AllocObjectID("cryptodev.Driver["+drv+"]"))
		vdev, e = eal.NewVDev(name, arg, socket)
		if e == nil {
			break
		}
		fmt.Fprintf(&drvErrors, "%s: %v; ", drv, e)
	}
	if vdev == nil {
		return nil, errors.New(drvErrors.String())
	}

	if cd, e = New(vdev, cfg); e != nil {
		return nil, e
	}
	return cd, nil
}
