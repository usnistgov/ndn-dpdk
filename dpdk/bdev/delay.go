package bdev

import (
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// DelayConfig configures Delay bdev.
type DelayConfig struct {
	AvgReadLatency  time.Duration
	P99ReadLatency  time.Duration
	AvgWriteLatency time.Duration
	P99WriteLatency time.Duration
}

// Delay represents a delay block device.
type Delay struct {
	*Info
}

var _ DeviceCloser = (*Delay)(nil)

// Close destroys this block device.
// The inner device is not closed.
func (device *Delay) Close() error {
	return deleteByName("bdev_delay_delete", device.Name())
}

// NewDelay creates a delay block device.
func NewDelay(inner Device, cfg DelayConfig) (device *Delay, e error) {
	args := struct {
		Name            string `json:"name"`
		BaseBdevName    string `json:"base_bdev_name"`
		AvgReadLatency  int    `json:"avg_read_latency"`
		P99ReadLatency  int    `json:"p99_read_latency"`
		AvgWriteLatency int    `json:"avg_write_latency"`
		P99WriteLatency int    `json:"p99_write_latency"`
	}{
		Name:            eal.AllocObjectID("bdev.Delay"),
		BaseBdevName:    inner.DevInfo().Name(),
		AvgReadLatency:  int(cfg.AvgReadLatency.Microseconds()),
		P99ReadLatency:  int(cfg.P99ReadLatency.Microseconds()),
		AvgWriteLatency: int(cfg.AvgWriteLatency.Microseconds()),
		P99WriteLatency: int(cfg.P99WriteLatency.Microseconds()),
	}
	var name string
	if e = spdkenv.RPC("bdev_delay_create", args, &name); e != nil {
		return nil, e
	}
	return &Delay{mustFind(name)}, nil
}
