package ethvdev

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

const memifChownDeadline = 5 * time.Second

var memifCoexist = make(memiftransport.CoexistMap)

// NewMemif creates a net_memif device.
func NewMemif(loc memiftransport.Locator) (ethdev.EthDev, error) {
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memiftransport.Locator.ToVDevArgs %w", e)
	}

	if e := memifCoexist.Check(loc); e != nil {
		return nil, e
	}
	memifCheckSocket(loc.Role, loc.SocketName)

	name := "net_memif" + eal.AllocObjectID("ethvdev.Memif")
	dev, e := New(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethvdev.New %w", e)
	}

	chownTimeout, chownCancel := context.WithTimeout(context.TODO(), memifChownDeadline)
	if _, ok := memifCoexist[loc.SocketName]; !ok && loc.SocketOwner != nil {
		go memifChown(chownTimeout, chownCancel, loc.SocketName, *loc.SocketOwner)
	} else {
		chownCancel()
	}

	memifCoexist.Add(loc)
	ethdev.OnDetach(dev, func() {
		chownCancel()
		memifCoexist.Remove(loc)
	})
	return dev, nil
}

func memifCheckSocket(role memiftransport.Role, socketName string) {
	logEntry := logger.With(zap.String("socketName", socketName))
	st, e := os.Stat(socketName)
	switch role {
	case memiftransport.RoleServer:
		if e := os.MkdirAll(path.Dir(socketName), 0777); e != nil {
			logEntry.Warn("cannot create directory containing socket file", zap.Error(e))
		}
		if e == nil && st.Mode().Type()&os.ModeSocket == 0 {
			logEntry.Warn("socket file already exists but it is not a Unix socket; if ethdev creation fails, delete the socket file")
		}
	case memiftransport.RoleClient:
		if e != nil || st.Mode().Type()&os.ModeSocket == 0 {
			logEntry.Warn("socket file does not exist or it is not a Unix socket; if ethdev creation fails, ensure the memif server is running")
		}
	}
}

func memifChown(ctx context.Context, cancel context.CancelFunc, socketName string, owner [2]int) {
	defer cancel()
	uid, gid := owner[0], owner[1]
	logEntry := logger.With(zap.String("socketName", socketName), zap.Int("uid", uid), zap.Int("gid", gid))
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

WAIT:
	for {
		select {
		case <-ctx.Done():
			logEntry.Warn("memif SocketOwner socket file did not show up within deadline", zap.Duration("deadline", memifChownDeadline))
			return
		case <-tick.C:
			_, e := os.Stat(socketName)
			switch {
			case e == nil:
				break WAIT
			case errors.Is(e, fs.ErrNotExist):
				continue WAIT
			default:
				logEntry.Warn("memif SocketOwner stat error", zap.Error(e))
				return
			}
		}
	}

	if e := unix.Chown(socketName, owner[0], owner[1]); e != nil {
		logEntry.Warn("memif SocketOwner chown error", zap.Error(e))
		return
	}
	logEntry.Info("memif SocketOwner changed")
}
