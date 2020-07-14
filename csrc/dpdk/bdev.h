#ifndef NDNDPDK_DPDK_BDEV_H
#define NDNDPDK_DPDK_BDEV_H

/** @file */

#include "mbuf.h"
#include <spdk/bdev.h>

/**
 * @brief Maximum number of segments in packet acceptable to
 *        SpdkBdev_ReadPacket and SpdkBdev_WritePacket.
 */
#define SPDK_BDEV_MAX_MBUF_SEGS 32

/** @brief Allocate filler buffer for scatter gather functions. */
void
SpdkBdev_InitFiller();

/** @brief Read block device into mbuf via scatter gather list. */
int
SpdkBdev_ReadPacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                    uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                    spdk_bdev_io_completion_cb cb, void* ctx);

/** @brief Write block device from mbuf via scatter gather list. */
int
SpdkBdev_WritePacket(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, struct rte_mbuf* pkt,
                     uint64_t blockOffset, uint64_t blockCount, uint32_t blockSize,
                     spdk_bdev_io_completion_cb cb, void* ctx);

#endif // NDNDPDK_SPDK_BDEV_H
