#ifndef NDNDPDK_DPDK_BDEV_H
#define NDNDPDK_DPDK_BDEV_H

/** @file */

#include "mbuf.h"
#include <spdk/bdev.h>

enum
{
  /** @brief Maximum packet segments acceptable to Bdev_ReadPacket and Bdev_WritePacket. */
  BdevMaxMbufSegs = 32,

  BdevFillerLen_ = 65536,
};

extern void* BdevFillers_;

typedef struct Bdev Bdev;
typedef struct BdevRequest BdevRequest;

typedef void (*BdevRequestCb)(BdevRequest* req, int res);

/** @brief Bdev I/O request. */
struct BdevRequest
{
  BdevRequestCb cb;
  struct iovec iov[BdevMaxMbufSegs + 1];
};

typedef int (*Bdev_PrepareIovecFunc)(Bdev* bd, BdevRequest* req, struct rte_mbuf* pkt,
                                     uint64_t blockCount, void* filler);

/** @brief Block device and related information. */
struct Bdev
{
  struct spdk_bdev_desc* desc;
  uint32_t blockSizeMinus1;
  uint32_t blockSizeLog2;
};

/**
 * @brief Determine number of blocks needed to store a packet.
 *
 * CEIL(pkt->pkt_len / bd->blockSize)
 */
__attribute__((nonnull)) static __rte_always_inline uint32_t
Bdev_ComputeBlockCount(Bdev* bd, struct rte_mbuf* pkt)
{
  return (pkt->pkt_len >> bd->blockSizeLog2) + (int)((pkt->pkt_len & bd->blockSizeMinus1) != 0);
}

/** @brief Read block device into mbuf via scatter gather list. */
__attribute__((nonnull)) void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                BdevRequestCb cb, BdevRequest* req);

/** @brief Write block device from mbuf via scatter gather list. */
__attribute__((nonnull)) void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                 BdevRequestCb cb, BdevRequest* req);

#endif // NDNDPDK_SPDK_BDEV_H
