#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

__attribute__((nonnull)) static inline void
GetData_Finalize(DiskStoreRequest* req)
{
  rte_mempool_put(req->store->mp, req);
}

__attribute__((nonnull)) static void
GetData_Fail(DiskStore* store, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  if (interest->diskData != NULL) {
    rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
    interest->diskData = NULL;
  }

  store->getDataCb(npkt, store->getDataCtx);
}

__attribute__((nonnull)) static void
GetData_End(BdevRequest* breq, int res)
{
  DiskStoreRequest* req = container_of(breq, DiskStoreRequest, breq);
  if (unlikely(res != 0)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, req->slotID, req->npkt, res);
    GetData_Fail(req->store, req->npkt);
    goto FINISH;
  }

  PInterest* interest = Packet_GetInterestHdr(req->npkt);
  Mbuf_SetTimestamp(Packet_ToMbuf(interest->diskData), rte_get_tsc_cycles());
  if (unlikely(!Packet_Parse(interest->diskData)) ||
      unlikely(Packet_GetType(interest->diskData) != PktData)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("not-Data"), req->slotID,
           req->npkt);
    GetData_Fail(req->store, req->npkt);
    goto FINISH;
  }

  N_LOGD("GetData success slot=%" PRIu64 " npkt=%p", req->slotID, req->npkt);
  req->store->getDataCb(req->npkt, req->store->getDataCtx);

FINISH:
  GetData_Finalize(req);
}

__attribute__((nonnull)) static void
GetData_Begin(void* req0)
{
  DiskStoreRequest* req = (DiskStoreRequest*)req0;
  NDNDPDK_ASSERT(req->store->ch != NULL);

  uint64_t blockOffset = req->slotID * req->store->nBlocksPerSlot;
  PInterest* interest = Packet_GetInterestHdr(req->npkt);
  Bdev_ReadPacket(&req->store->bdev, req->store->ch, Packet_ToMbuf(interest->diskData), blockOffset,
                  GetData_End, &req->breq);
}

void
DiskStore_GetData(DiskStore* store, uint64_t slotID, Packet* npkt, struct rte_mbuf* dataBuf)
{
  NDNDPDK_ASSERT(slotID > 0);

  DiskStoreRequest* req = NULL;
  int res = rte_mempool_get(store->mp, (void**)&req);
  if (unlikely(res != 0)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("alloc-err"), slotID, npkt);
    GetData_Fail(store, npkt);
    return;
  }

  PInterest* interest = Packet_GetInterestHdr(npkt);
  interest->diskSlot = slotID;
  interest->diskData = Packet_FromMbuf(dataBuf);

  N_LOGD("GetData request slot=%" PRIu64 " npkt=%p", slotID, npkt);
  req->store = store;
  req->slotID = slotID;
  req->npkt = npkt;

  res = spdk_thread_send_msg(store->th, GetData_Begin, req);
  if (unlikely(res != 0)) {
    N_LOGW("GetData error spdk_thread_send_msg slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID,
           npkt, res);
    GetData_Fail(store, npkt);
    GetData_Finalize(req);
  }
}
