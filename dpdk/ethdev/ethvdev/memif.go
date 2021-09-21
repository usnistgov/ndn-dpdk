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

var memifCoexist = memiftransport.NewCoexistMap()

// NewMemif creates a net_memif device.
func NewMemif(loc memiftransport.Locator) (ethdev.EthDev, error) {
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memiftransport.Locator.ToVDevArgs %w", e)
	}

	if e := memifCoexist.Check(loc); e != nil {
		return nil, e
	}
	isFirst := !memifCoexist.Has(loc.SocketName)
	memifCheckSocket(loc.Role, loc.SocketName, isFirst)

	name := "net_memif" + eal.AllocObjectID("ethvdev.Memif")
	dev, e := New(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethvdev.New %w", e)
	}

	var chownCancel context.CancelFunc
	if isFirst && loc.Role == memiftransport.RoleServer && loc.SocketOwner != nil {
		timeout, cancel := context.WithTimeout(context.TODO(), memifChownDeadline)
		go memifChown(timeout, cancel, loc.SocketName, *loc.SocketOwner)
		chownCancel = cancel
	}

	memifCoexist.Add(loc)
	ethdev.OnDetach(dev, func() {
		if chownCancel != nil {
			chownCancel()
		}
		memifCoexist.Remove(loc)
	})
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
		if isFirst {
			if ste == nil {
				if e := os.Remove(socketName); e != nil {
					logEntry.Warn("cannot delete dangling socket file; if ethdev creation fails, manually delete the socket file", zap.Error(e))
				} else {
					logEntry.Debug("deleted dangling socket file")
				}
			} else if e := os.MkdirAll(path.Dir(socketName), 0777); e != nil {
				logEntry.Warn("cannot create directory containing socket file", zap.Error(e))
			}
		} else if ste == nil && st.Mode().Type()&os.ModeSocket == 0 {
			logEntry.Warn("socket file exists but it is not a Unix socket; if ethdev creation fails, manually delete the socket file")
		}
	case memiftransport.RoleClient:
		if ste != nil || st.Mode().Type()&os.ModeSocket == 0 {
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
