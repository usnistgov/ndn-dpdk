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

extern void* BdevFiller_;

/** @brief Length and alignment descriptor of a stored packet. */
typedef struct BdevStoredPacket
{
  uint16_t pktLen;
  uint16_t saveTotal;
  uint16_t saveLen[BdevMaxMbufSegs];
  uint8_t headTail[BdevMaxMbufSegs]; // (headLen << 4) | (headLen + tailLen)
} BdevStoredPacket;

__attribute__((nonnull)) static __rte_always_inline void
BdevStoredPacket_Copy(BdevStoredPacket* dst, const BdevStoredPacket* src)
{
  if (src->pktLen == src->saveTotal) {
    dst->pktLen = src->pktLen;
    dst->saveTotal = src->saveTotal;
  } else {
    *dst = *src;
  }
}

typedef struct BdevRequest BdevRequest;
typedef void (*BdevRequestCb)(BdevRequest* req, int res);

/** @brief Bdev I/O request. */
struct BdevRequest
{
  struct rte_mbuf* pkt;
  BdevStoredPacket* sp;
  BdevRequestCb cb;
  struct iovec iov_[BdevMaxMbufSegs + 1];
};

/** @brief Block device and related information. */
typedef struct Bdev
{
  struct spdk_bdev_desc* desc;
  uint32_t blockSizeMinus1;
  uint32_t blockSizeLog2;
  uint32_t bufAlign;
  bool dwordAlign;
} Bdev;

/** @brief Determine number of blocks needed to store a packet. */
__attribute__((nonnull)) static __rte_always_inline uint32_t
Bdev_ComputeBlockCount(Bdev* bd, BdevStoredPacket* sp)
{
  // CEIL(saveTotal / bd->blockSize)
  return (sp->saveTotal >> bd->blockSizeLog2) + (int)((sp->saveTotal & bd->blockSizeMinus1) != 0);
}

/**
 * @brief Read block device into mbuf.
 * @pre This must be called in a SPDK thread.
 * @param ch an SPDK I/O channel associated with the bdev and the current SPDK thread.
 * @param req request context. All fields must be kept alive until @c req->cb is called.
 *
 * @c req->pkt must be a uniquely owned, unsegmented, direct mbuf with succifient dataroom.
 * @c req->pkt->data_off may be adjusted (either increased or decreased) to achieve proper
 * alignment as required by SPDK bdev driver.
 */
__attribute__((nonnull)) void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req);

/**
 * @brief Prepare writing according to bdev alignment requirements.
 * @param[in] pkt input packet; cannot be modified between this call and @c Bdev_WritePacket .
 * @param[out] sp stored packet alignment information, needed to later recover the packet.
 * @return whether success.
 */
__attribute__((nonnull)) bool
Bdev_WritePrepare(Bdev* bd, struct rte_mbuf* pkt, BdevStoredPacket* sp);

/**
 * @brief Write block device from mbuf via scatter gather list.
 * @pre This must be called in a SPDK thread.
 * @param ch an SPDK I/O channel associated with the bdev and the current SPDK thread.
 * @param req request context. All fields except @c req->sp must be kept alive until @c req->cb
 *            is called.
 *
 * @c req->pkt->pkt_len determines write length. Some headroom and tailroom in each mbuf segment
 * may be written to disk to achieve proper alignment as required by SPDK bdev driver, but they
 * will not appear in readback if the same @c BdevStorePacket is passed.
 */
__attribute__((nonnull)) void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req);

#endif // NDNDPDK_SPDK_BDEV_H
