#ifndef NDNDPDK_DPDK_BDEV_H
#define NDNDPDK_DPDK_BDEV_H

/** @file */

#include "bdev-enum.h"
#include "mbuf.h"
#include <spdk/bdev.h>

enum
{
  /** @brief Expected block size; other block sizes are not supported. */
  BdevBlockSize = 512,

  /** @brief Maximum packet segments acceptable to Bdev_ReadPacket and Bdev_WritePacket. */
  BdevMaxMbufSegs = 31,
};

extern uint8_t* BdevFiller_;

/** @brief Length and alignment descriptor of a stored packet. */
typedef struct BdevStoredPacket
{
  uint16_t pktLen;                   ///< original packet length
  uint16_t saveTotal;                ///< saved total length
  uint16_t saveLen[BdevMaxMbufSegs]; ///< per-segment saved length, zero denotes ending
  uint8_t headTail[BdevMaxMbufSegs]; ///< per-segment (headLen << 4) | (headLen + tailLen)
} BdevStoredPacket;

__attribute__((nonnull)) static __rte_always_inline void
BdevStoredPacket_Copy(BdevStoredPacket* restrict dst, const BdevStoredPacket* restrict src)
{
  if (src->pktLen == src->saveTotal) { // no head or tail
    goto COPY0;
  } else if (src->saveLen[1] == 0) { // single segment
    dst->headTail[0] = src->headTail[0];
    dst->headTail[1] = src->headTail[1];
  COPY0:
    dst->pktLen = src->pktLen;
    dst->saveTotal = src->saveTotal;
    dst->saveLen[0] = src->saveLen[0];
    dst->saveLen[1] = src->saveLen[1];
  } else { // multiple segments
    *dst = *src;
  }
}

/** @brief Determine number of blocks needed to store a packet. */
__attribute__((nonnull)) static inline uint32_t
BdevStoredPacket_ComputeBlockCount(BdevStoredPacket* sp)
{
  return SPDK_CEIL_DIV(sp->saveTotal, BdevBlockSize);
}

typedef struct BdevRequest BdevRequest;
typedef void (*BdevRequestCb)(BdevRequest* req, int res);

/** @brief Bdev I/O request. */
struct BdevRequest
{
  struct rte_mbuf* pkt;
  BdevStoredPacket* sp;
  BdevRequestCb cb;
  struct rte_mbuf* bounce_;
  struct iovec iov_[BdevMaxMbufSegs + 1];
};

/** @brief Block device and related information. */
typedef struct Bdev
{
  struct spdk_bdev_desc* desc;
  struct rte_mempool* bounceMp;
  uint32_t bufAlign;
  BdevWriteMode writeMode;
} Bdev;

/**
 * @brief Read block device into mbuf.
 * @pre This must be called in a SPDK thread.
 * @param ch an SPDK I/O channel associated with the bdev and the current SPDK thread.
 * @param req request context. All fields must be kept alive until @c req->cb is called.
 *
 * @c req->pkt must be a uniquely owned, unsegmented, direct mbuf with sufficient dataroom.
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
 * @param req request context. @c req->sp may be freed after this function returns. All other
 *            fields must be kept alive until @c req->cb is called.
 *
 * @c req->pkt->pkt_len determines write length. Some headroom and tailroom in each mbuf segment
 * may be written to disk to achieve proper alignment as required by SPDK bdev driver, but they
 * will not appear in readback if the same @c BdevStorePacket is passed.
 */
__attribute__((nonnull)) void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, uint64_t blockOffset, BdevRequest* req);

#endif // NDNDPDK_SPDK_BDEV_H
