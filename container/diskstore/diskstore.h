#ifndef NDN_DPDK_CONTAINER_DISKSTORE_DISKSTORE_H
#define NDN_DPDK_CONTAINER_DISKSTORE_DISKSTORE_H

/// \file

#include "../../ndn/packet.h"
#include "../../spdk/bdev.h"

#include <spdk/thread.h>

/** \brief Expected block size of the underlying block device.
 */
#define DISK_STORE_BLOCK_SIZE 512

/** \brief Disk-backed Data Store.
 */
typedef struct DiskStore
{
  struct spdk_thread* th;
  struct spdk_bdev_desc* bdev;
  struct spdk_io_channel* ch;
  struct rte_mempool* mp;
  uint64_t nBlocksPerSlot;
  uint32_t blockSize;
} DiskStore;

/** \brief Store a Data packet.
 *  \param slotId disk slot number; slot 0 cannot be used.
 *  \param npkt a Data packet. DiskStore takes ownership.
 *
 *  This function may be invoked on any thread, including non-SPDK thread.
 */
void DiskStore_PutData(DiskStore* store, uint64_t slotId, Packet* npkt);

/** \brief Retrieve a Data packet.
 *  \param slotId disk slot number.
 *  \param dataLen Data packet length.
 *  \param npkt an Interest packet. DiskStore takes ownership.
 *  \param reply where to return results.
 *
 *  This function asynchronously reads from a specified slot of the underlying
 *  disk, and parses the content as a Data packet. It then assigns
 *  <tt>Packet_GetInterestHdr(npkt)->diskSlotId</tt> and
 *  <tt>Packet_GetInterestHdr(npkt)->diskData</tt>, then enqueue \p npkt into
 *  \p reply.
 *
 *  This function may be invoked on any thread, including non-SPDK thread.
 */
void DiskStore_GetData(DiskStore* store, uint64_t slotId, uint16_t dataLen,
                       Packet* npkt, struct rte_ring* reply);

#endif // NDN_DPDK_CONTAINER_DISKSTORE_DISKSTORE_H
