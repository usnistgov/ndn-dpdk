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
  uint32_t bufAlign;
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

/**
 * @brief Read block device into mbuf via scatter gather list.
 * @pre This must be called in a SPDK thread.
 * @param ch an SPDK I/O channel associated with the bdev and the current SPDK thread.
 * @param pkt the packet to read packet into. It must be kept alive until @p cb is called.
 *
 * @c pkt->pkt_len determines read length. Since SPDK requires read length to be multiples of the
 * bdev block size, if @c pkt->pkt_len is not a multiple of block size, the tailroom of
 * @c rte_pktmbuf_lastseg(pkt) may be overwritten from what's stored on disk.
 *
 * Certain SPDK drivers require the buffer to have certain alignment; otherwise, SPDK would use
 * a bounce buffer and incur an additional copy. To reduce this overhead, if @p pkt is a uniquely
 * owned, unsegmented, direct mbuf with succifient dataroom, @c pkt->data_off may be adjusted
 * (either increased or decreased) to achieve proper alignment.
 */
__attribute__((nonnull)) void
Bdev_ReadPacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                BdevRequestCb cb, BdevRequest* req);

/**
 * @brief Write block device from mbuf via scatter gather list.
 * @pre This must be called in a SPDK thread.
 * @param ch an SPDK I/O channel associated with the bdev and the current SPDK thread.
 * @param pkt the packet to write packet from. It must be kept alive until @p cb is called.
 *
 * @c pkt->pkt_len determines write length. Since SPDK requires read length to be multiples of the
 * bdev block size, if @c pkt->pkt_len is not a multiple of block size, the tailroom of
 * @c rte_pktmbuf_lastseg(pkt) may be written to disk.
 */
__attribute__((nonnull)) void
Bdev_WritePacket(Bdev* bd, struct spdk_io_channel* ch, struct rte_mbuf* pkt, uint64_t blockOffset,
                 BdevRequestCb cb, BdevRequest* req);

#endif // NDNDPDK_SPDK_BDEV_H
