package bdev

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

var errorIOType = map[IOType]string{
	IORead:  "read",
	IOWrite: "write",
	IOUnmap: "unmap",
}

// ErrorInjection represents an error injection block device.
type ErrorInjection struct {
	*Info
}

var _ DeviceCloser = (*ErrorInjection)(nil)

// Close destroys this block device.
// The inner device is not closed.
func (device *ErrorInjection) Close() error {
	return deleteByName("bdev_error_delete", device.Name())
}

// Inject injects some errors.
func (device *ErrorInjection) Inject(ioType IOType, count int) error {
	args := struct {
		Name      string `json:"name"`
		IOType    string `json:"io_type"`
		ErrorType string `json:"error_type"`
		Num       int    `json:"num"`
	}{
		Name:      device.Name(),
		IOType:    errorIOType[ioType],
		ErrorType: "failure",
		Num:       count,
	}
	if args.IOType == "" {
		return errors.New("unsupported IOType")
	}
	var ok bool
	return spdkenv.RPC("bdev_error_inject_error", args, &ok)
}

// NewErrorInjection creates an error injection block device.
func NewErrorInjection(inner Device) (device *ErrorInjection, e error) {
	args := struct {
		BaseName string `json:"base_name"`
	}{
		BaseName: inner.DevInfo().Name(),
	}
	var ok bool
	if e = spdkenv.RPC("bdev_error_create", args, &ok); e != nil {
		return nil, e
	}
	return &ErrorInjection{mustFind("EE_" + args.BaseName)}, nil
}
