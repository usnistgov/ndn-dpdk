#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

static_assert((int)SPDK_BDEV_MAX_MBUF_SEGS >= (int)LpMaxFragments, "");

/** @brief Parameters related to PutData, stored over PData.digest field. */
typedef struct PutDataRequest
{
  DiskStore* store;
  uint64_t slotID;
} PutDataRequest;
static_assert(sizeof(PutDataRequest) <= sizeof(((PData*)(NULL))->digest), "");

__attribute__((nonnull)) static void
PutData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PData* data = Packet_GetDataHdr(npkt);
  PutDataRequest* req = (PutDataRequest*)&data->digest[0];
  uint64_t slotID = req->slotID;

  if (likely(success)) {
    N_LOGD("PutData success slot=%" PRIu64 " npkt=%p", slotID, npkt);
  } else {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("io-err"), slotID, npkt);
  }

  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  spdk_bdev_free_io(io);
}

__attribute__((nonnull)) static void
PutData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PData* data = Packet_GetDataHdr(npkt);
  PutDataRequest* req = (PutDataRequest*)&data->digest[0];
  DiskStore* store = req->store;
  uint64_t slotID = req->slotID;

  if (unlikely(store->ch == NULL)) {
    store->ch = spdk_bdev_get_io_channel(store->bdev);
    if (unlikely(store->ch == NULL)) {
      N_LOGW("PutData no I/O channel" N_LOG_ERROR("spdk_bdev_get_io_channel"));
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      return;
    }
  }

  uint64_t blockOffset = DiskStore_ComputeBlockOffset_(store, slotID);
  uint64_t blockCount = DiskStore_ComputeBlockCount_(store, npkt);
  int res = SpdkBdev_WritePacket(store->bdev, store->ch, Packet_ToMbuf(npkt), blockOffset,
                                 blockCount, store->blockSize, PutData_End, (uintptr_t)npkt);
  if (unlikely(res != 0)) {
    N_LOGW("PutData write error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID, npkt, res);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
  }
}

void
DiskStore_PutData(DiskStore* store, uint64_t slotID, Packet* npkt)
{
  NDNDPDK_ASSERT(slotID > 0);
  uint64_t blockCount = DiskStore_ComputeBlockCount_(store, npkt);
  if (unlikely(blockCount > store->nBlocksPerSlot)) {
    N_LOGW("PutData slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("packet-too-long"), slotID, npkt);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  PData* data = Packet_GetDataHdr(npkt);
  data->hasDigest = false;
  PutDataRequest* req = (PutDataRequest*)&data->digest[0];
  req->store = store;
  req->slotID = slotID;
  spdk_thread_send_msg(store->th, PutData_Begin, npkt);
}
