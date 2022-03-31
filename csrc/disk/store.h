#ifndef NDNDPDK_DISK_STORE_H
#define NDNDPDK_DISK_STORE_H

/** @file */

#include "../dpdk/bdev.h"
#include "../dpdk/hashtable.h"
#include "../dpdk/spdk-thread.h"
#include "../ndni/packet.h"

/** @brief Expected block size of the underlying block device. */
#define DiskStore_BlockSize 512

typedef struct DiskStore DiskStore;

/** @brief DiskStore compact request context. */
typedef struct DiskStoreSlimRequest
{
  Packet* next;
  DiskStore* store;
  uint64_t slotID;
} DiskStoreSlimRequest;

/** @brief DiskStore request context. */
typedef struct DiskStoreRequest
{
  DiskStoreSlimRequest s;
  union
  {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };
  BdevRequest breq;
} DiskStoreRequest;

/**
 * @brief DiskStore_GetData completion callback.
 * @param npkt Interest packet.
 * @param ctx @c store->getDataCtx .
 *
 * This function is invoked on @c store->th thread.
 */
typedef void (*DiskStore_GetDataCb)(Packet* npkt, uintptr_t ctx);

/** @brief Disk-backed Data packet store. */
struct DiskStore
{
  Bdev bdev;
  uint64_t nBlocksPerSlot;
  struct rte_hash* requestHt;
  DiskStoreRequest* requestArray;
  struct spdk_thread* th;
  struct spdk_io_channel* ch;
  DiskStore_GetDataCb getDataCb;
  uintptr_t getDataCtx;

  uint64_t nPutDataBegin;
  uint64_t nPutDataFinish[2]; // 0=failure, 1=success
  uint64_t nGetDataBegin;
  uint64_t nGetDataReuse;
  uint64_t nGetDataSuccess;
  uint64_t nGetDataFailure;
};

/**
 * @brief Prepare to store a Data packet.
 * @return whether success.
 */
__attribute__((nonnull)) static inline bool
DiskStore_PutPrepare(DiskStore* store, Packet* npkt, BdevStoredPacket* sp)
{
  return Bdev_WritePrepare(&store->bdev, Packet_ToMbuf(npkt), sp);
}

/**
 * @brief Store a Data packet.
 * @param slotID disk slot number; slot 0 cannot be used.
 * @param npkt a Data packet. DiskStore takes ownership.
 * @param sp output of a successful @c DiskStore_PrepareData .
 *
 * This function may be invoked on any thread, including non-SPDK thread.
 */
__attribute__((nonnull)) void
DiskStore_PutData(DiskStore* store, uint64_t slotID, Packet* npkt, BdevStoredPacket* sp);

/**
 * @brief Retrieve a Data packet.
 * @param slotID disk slot number.
 * @param npkt an Interest packet. DiskStore takes ownership until callback.
 * @param dataBuf a uniquely owned, unsegmented, direct mbuf for Data packet.
 *                DiskStore takes ownership until callback.
 * @param sp same @c BdevStoredPacket used during PutData, will be copied.
 *
 * This function asynchronously reads from a specified slot of the underlying disk, and parses
 * the content as a Data packet.
 * Upon success, it assigns @c interest->diskSlot and @c interest->diskData .
 * Upon failure, it assigns @c interest->diskSlot and clears @c interest->diskData .
 * It then calls @c store->getDataCb with the @p npkt .
 *
 * This function may be invoked on any thread, including non-SPDK thread.
 */
__attribute__((nonnull)) void
DiskStore_GetData(DiskStore* store, uint64_t slotID, Packet* npkt, struct rte_mbuf* dataBuf,
                  BdevStoredPacket* sp);

#endif // NDNDPDK_DISK_STORE_H
