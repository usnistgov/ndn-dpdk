package bdev

import "strconv"

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_DPDK_BDEV_ENUM_H -out=../../csrc/dpdk/bdev-enum.h .

// WriteMode indicates memory alignment requirements of writev request.
type WriteMode int

// WriteMode values.
const (
	WriteModeSimple WriteMode = iota
	WriteModeDwordAlign
	WriteModeContiguous
	_ = "enumgen:BdevWriteMode:BdevWriteMode:WriteMode"
)

func (wm WriteMode) String() string {
	switch wm {
	case WriteModeSimple:
		return "simple"
	case WriteModeDwordAlign:
		return "dword-align"
	case WriteModeContiguous:
		return "contiguous"
	}
	return strconv.Itoa(int(wm))
}

type withWriteMode interface {
	writeMode() WriteMode
}

// OverrideWriteMode forces a specific WriteMode for unit testing.
func OverrideWriteMode(device Device, wm WriteMode) Device {
	return deviceWriteModeOverride{
		Device: device,
		wm:     wm,
	}
}

type deviceWriteModeOverride struct {
	Device
	wm WriteMode
}

func (device deviceWriteModeOverride) writeMode() WriteMode {
	return device.wm
}
