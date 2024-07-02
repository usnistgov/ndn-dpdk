package ethdev

import (
	"fmt"
	"os"
	"path"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"go.uber.org/zap"
)

var memifCoexist = memiftransport.NewCoexistMap()

// NewMemif creates a net_memif device.
func NewMemif(loc memiftransport.Locator) (EthDev, error) {
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memiftransport.Locator.ToVDevArgs %w", e)
	}

	if e := memifCoexist.Check(loc); e != nil {
		return nil, e
	}
	isFirst := !memifCoexist.Has(loc.SocketName)
	memifCheckSocket(loc.Role, loc.SocketName, isFirst)

	name := "net_memif" + eal.AllocObjectID("ethdev.Memif")
	dev, e := NewVDev(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethdev.NewVDev(%s) %w", name, e)
	}

	memifCoexist.Add(loc)
	OnClose(dev, func() { memifCoexist.Remove(loc) })
	return dev, nil
}

func memifCheckSocket(role memiftransport.Role, socketName string, isFirst bool) {
	logEntry := logger.With(
		zap.String("role", string(role)),
		zap.String("socketName", socketName),
		zap.Bool("isFirst", isFirst),
	)

	st, ste := os.Stat(socketName)
	switch role {
	case memiftransport.RoleServer:
		if isFirst { // first interface: socket should not exist, directory should exist
			switch {
			case ste != nil: // socket does not exist, directory may or may not exist
				if e := os.MkdirAll(path.Dir(socketName), 0777); e != nil {
					logEntry.Warn("cannot create directory containing socket file", zap.Error(e))
				}
			case st.Mode().Type()&os.ModeSocket != 0: // socket exist
				if e := os.Remove(socketName); e != nil {
					logEntry.Warn("cannot delete dangling socket file; if ethdev creation fails, manually delete the socket file", zap.Error(e))
				} else {
					logEntry.Debug("deleted dangling socket file")
				}
			default: // file exist but not a socket
				logEntry.Warn("file exists but is not a socket; if ethdev creation fails, manually delete the socket file")
			}
		}
		// not checking for non-first interface: if there's a problem, first interface should fail to create
	case memiftransport.RoleClient: // socket should exist
		if ste != nil || st.Mode().Type()&os.ModeSocket == 0 {
			logEntry.Warn("socket file does not exist or it is not a Unix socket; if ethdev creation fails, ensure the memif server is running")
		}
	}
}
