package ethdev

/*
#include "../../csrc/core/common.h"
#include <rte_pmd_i40e.h>
#cgo LDFLAGS: -lrte_net_i40e
*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

// DdpProfile represents a Dynamic Device Personalization profile.
type DdpProfile struct {
	pkg    []byte
	info   C.struct_rte_pmd_i40e_profile_info
	zapPkg zap.Field
}

func (dp *DdpProfile) process(dev EthDev, buf []byte, op C.enum_rte_pmd_i40e_package_op, act string) error {
	bufC, bufSize := (*C.uint8_t)(&buf[0]), C.uint32_t(len(buf))
	if res := C.rte_pmd_i40e_process_ddp_package(dev.(ethDev).cID(), bufC, bufSize, op); res != 0 {
		e := eal.MakeErrno(res)
		logger.Error(act+" DDP package error",
			dev.ZapField("port"),
			dp.zapPkg,
			zap.Error(e),
		)
		return fmt.Errorf("rte_pmd_i40e_process_ddp_package %w", e)
	}

	logger.Info(act+" DDP package success",
		dev.ZapField("port"),
		dp.zapPkg,
	)
	return nil
}

// Upload adds the DDP profile to an i40e device.
// The returned rollback function removes the DDP profile from the device.
func (dp *DdpProfile) Upload(dev EthDev) (rollback func() error, e error) {
	// OP_WR_ADD modifies the buffer; OP_WR_DEL expects the modified buffer
	buf := slices.Clone(dp.pkg)
	if e = dp.process(dev, buf, C.RTE_PMD_I40E_PKG_OP_WR_ADD, "upload"); e != nil {
		return nil, e
	}
	return func() error {
		return dp.process(dev, buf, C.RTE_PMD_I40E_PKG_OP_WR_DEL, "rollback")
	}, nil
}

// OpenDdpProfile opens a DDP profile from /lib/firmware/intel/i40e/ddp/{}.pkg .
func OpenDdpProfile(name string) (dp *DdpProfile, e error) {
	filename := path.Join("/lib/firmware/intel/i40e/ddp", name+".pkg")
	logEntry := logger.With(zap.String("filename", filename))
	file, e := os.Open(filename)
	if e != nil {
		logEntry.Warn("open DDP profile error", zap.Error(e))
		return nil, e
	}
	defer file.Close()

	dp = &DdpProfile{}
	dp.pkg, e = io.ReadAll(file)
	if e != nil {
		logEntry.Warn("read DDP profile error", zap.Error(e))
		return nil, e
	}

	if res := C.rte_pmd_i40e_get_ddp_info((*C.uint8_t)(&dp.pkg[0]), C.uint32_t(len(dp.pkg)),
		(*C.uint8_t)(unsafe.Pointer(&dp.info)), C.uint32_t(unsafe.Sizeof(dp.info)),
		C.RTE_PMD_I40E_PKG_INFO_GLOBAL_HEADER); res != 0 {
		e = eal.MakeErrno(res)
		logEntry.Warn("parse DDP profile error", zap.Error(e))
		return nil, fmt.Errorf("rte_pmd_i40e_get_ddp_info %w", e)
	}

	dp.zapPkg = zap.Dict("pkg",
		zap.Uint32("track-id", uint32(dp.info.track_id)),
		zap.String("name", C.GoString((*C.char)(unsafe.Pointer(&dp.info.name[0])))),
		zap.String("version", fmt.Sprintf("%d.%d.%d.%d",
			dp.info.version.major, dp.info.version.minor, dp.info.version.update, dp.info.version.draft)),
	)
	return dp, nil
}
