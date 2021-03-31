package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

// NewVDev creates an EthDev from a virtual device.
// The VDev will be destroyed when the EthDev is stopped and detached.
func NewVDev(name, args string, socket eal.NumaSocket) (EthDev, error) {
	vdev, e := eal.NewVDev(name, args, socket)
	if e != nil {
		return nil, e
	}

	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	var port C.uint16_t
	res := C.rte_eth_dev_get_port_by_name(nameC, &port)
	if res != 0 {
		panic("unexpected eth_dev_get_port_by_name error")
	}

	dev := ethDev(port)
	OnDetach(dev, func() {
		e := vdev.Close()
		logger.Debug("close vdev",
			zap.Int("id", dev.ID()),
			zap.Error(e),
		)
	})
	return dev, nil
}
