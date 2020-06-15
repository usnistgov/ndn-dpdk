package spdk

/*
#include "../csrc/core/common.h"
#include <spdk/accel_engine.h>
*/
import "C"
import (
	"fmt"
	"sync"

	"ndn-dpdk/dpdk/eal"
)

var initAccelEngineOnce sync.Once

func initAccelEngine() {
	initAccelEngineOnce.Do(func() {
		MainThread.Call(func() { C.spdk_accel_engine_initialize() })
	})
}

type bdevMallocCreateArgs struct {
	BlockSize int `json:"block_size"`
	NumBlocks int `json:"num_blocks"`
}

// Create Malloc block device.
func NewMallocBdev(blockSize int, nBlocks int) (bdi BdevInfo, e error) {
	initAccelEngine() // Malloc bdev depends on accelerator engine
	var args bdevMallocCreateArgs
	args.BlockSize = blockSize
	args.NumBlocks = nBlocks
	var name string
	if e = RpcCall("bdev_malloc_create", args, &name); e != nil {
		return BdevInfo{}, e
	}
	return mustFindBdev(name), nil
}

type bdevMallocDeleteArgs struct {
	Name string `json:"name"`
}

// Destroy Malloc block device.
func DestroyMallocBdev(bdi BdevInfo) (e error) {
	var args bdevMallocDeleteArgs
	args.Name = bdi.GetName()
	var ok bool
	return RpcCall("bdev_malloc_delete", args, &ok)
}

var lastAioBdevId int

type bdevAioCreateArgs struct {
	Name      string `json:"name"`
	Filename  string `json:"filename"`
	BlockSize int    `json:"block_size,omitempty"`
}

// Create Linux AIO block device.
func NewAioBdev(filename string, blockSize int) (bdi BdevInfo, e error) {
	var args bdevAioCreateArgs
	lastAioBdevId++
	args.Name = fmt.Sprintf("Aio%d", lastAioBdevId)
	args.Filename = filename
	args.BlockSize = blockSize
	var name string
	if e = RpcCall("bdev_aio_create", args, &name); e != nil {
		return BdevInfo{}, e
	}
	return mustFindBdev(name), nil
}

type bdevAioDeleteArgs struct {
	Name string `json:"name"`
}

// Destroy Linux AIO block device.
func DestroyAioBdev(bdi BdevInfo) (e error) {
	var args bdevAioDeleteArgs
	args.Name = bdi.GetName()
	var ok bool
	return RpcCall("bdev_aio_delete", args, &ok)
}

func makeNvmeName(pciAddr eal.PciAddress) string {
	return fmt.Sprintf("nvme%02x%02x%01x", pciAddr.Bus, pciAddr.Devid, pciAddr.Function)
}

type bdevNvmeAttachController struct {
	Name   string `json:"name"`
	TrType string `json:"trtype"`
	TrAddr string `json:"traddr"`
}

func AttachNvmeBdevs(pciAddr eal.PciAddress) (bdis []BdevInfo, e error) {
	var args bdevNvmeAttachController
	args.Name = makeNvmeName(pciAddr)
	args.TrType = "pcie"
	args.TrAddr = pciAddr.String()

	var namespaces []string
	if e = RpcCall("bdev_nvme_attach_controller", args, &namespaces); e != nil {
		return nil, e
	}

	for _, namespace := range namespaces {
		bdis = append(bdis, mustFindBdev(namespace))
	}
	return bdis, nil
}

type bdevNvmeDetachControllerArgs struct {
	Name string `json:"name"`
}

func DetachNvmeBdevs(pciAddr eal.PciAddress) (e error) {
	var args bdevNvmeDetachControllerArgs
	args.Name = makeNvmeName(pciAddr)
	var ok bool
	return RpcCall("bdev_nvme_detach_controller", args, &ok)
}
