#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

static_assert((int)BdevMaxMbufSegs >= (int)LpMaxFragments, "");

__attribute__((nonnull)) static inline void
PutData_Finalize(DiskStoreRequest* req)
{
  rte_pktmbuf_free(req->pkt);
  rte_mempool_put(req->store->mp, req);
}

__attribute__((nonnull)) static void
PutData_End(BdevRequest* breq, int res)
{
  DiskStoreRequest* req = container_of(breq, DiskStoreRequest, breq);
  if (likely(res == 0)) {
    N_LOGD("PutData success slot=%" PRIu64 " npkt=%p", req->slotID, req->npkt);
  } else {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, req->slotID, req->npkt, res);
  }
  PutData_Finalize(req);
}

__attribute__((nonnull)) static void
PutData_Begin(void* req0)
{
  DiskStoreRequest* req = (DiskStoreRequest*)req0;
  uint64_t blockOffset = req->slotID * req->store->nBlocksPerSlot;
  Bdev_WritePacket(&req->store->bdev, req->store->ch, req->pkt, blockOffset, PutData_End,
                   &req->breq);
}

void
DiskStore_PutData(DiskStore* store, uint64_t slotID, Packet* npkt)
{
  NDNDPDK_ASSERT(slotID > 0);

  uint32_t blockCount = Bdev_ComputeBlockCount(&store->bdev, Packet_ToMbuf(npkt));
  if (unlikely(blockCount > store->nBlocksPerSlot)) {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("packet-too-long"), slotID, npkt);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  DiskStoreRequest* req = NULL;
  int res = rte_mempool_get(store->mp, (void**)&req);
  if (unlikely(res != 0)) {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("alloc-err"), slotID, npkt);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  N_LOGD("PutData request slot=%" PRIu64 " npkt=%p", slotID, npkt);
  req->store = store;
  req->slotID = slotID;
  req->npkt = npkt;
  res = spdk_thread_send_msg(store->th, PutData_Begin, req);
  if (unlikely(res != 0)) {
    N_LOGW("PutData error spdk_thread_send_msg slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID,
           npkt, res);
    PutData_Finalize(req);
  }
}
