#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

static_assert((int)BdevMaxMbufSegs == (int)LpMaxFragments, "");

__attribute__((nonnull, returns_nonnull)) static __rte_always_inline DiskStoreSlimRequest*
DiskStoreSlimRequest_FromData(PData* data)
{
  static_assert(sizeof(DiskStoreSlimRequest) <= sizeof(data->helperScratch), "");
  return (void*)data->helperScratch;
}

__attribute__((nonnull, returns_nonnull)) static __rte_always_inline DiskStoreSlimRequest*
DiskStoreSlimRequest_FromPacket(Packet* npkt)
{
  switch (Packet_GetType(npkt)) {
    case PktData: {
      PData* data = Packet_GetDataHdr(npkt);
      return DiskStoreSlimRequest_FromData(data);
    }
    case PktInterest: {
      PInterest* interest = Packet_GetInterestHdr(npkt);
      return rte_mbuf_to_priv(Packet_ToMbuf(interest->diskData));
    }
    default:
      NDNDPDK_ASSERT(false);
      return (void*)npkt;
  }
}

__attribute__((nonnull)) static inline void
PutData_Finish(DiskStore* store, Packet* npkt, int res)
{
  ++store->nPutDataFinish[(int)(res == 0)];
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

__attribute__((nonnull)) static void
PutData_End(BdevRequest* breq, int res);

__attribute__((nonnull)) static inline void
PutData_Begin(DiskStore* store, DiskStoreRequest* req, Packet* npkt, uint64_t slotID)
{
  ++store->nPutDataBegin;
  N_LOGD("PutData begin slot=%" PRIu64 " npkt=%p", slotID, npkt);
  BdevStoredPacket sp;
  Bdev_WritePrepare(&store->bdev, req->pkt, &sp);
  uint64_t blockOffset = slotID * store->nBlocksPerSlot;
  req->breq.pkt = req->pkt;
  req->breq.sp = &sp;
  req->breq.cb = PutData_End;
  Bdev_WritePacket(&store->bdev, store->ch, blockOffset, &req->breq);
  NULLize(req->breq.sp);
}

__attribute__((nonnull)) static inline void
GetData_Finish(DiskStore* store, Packet* npkt, int res)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  if (likely(res == 0)) {
    ++store->nGetDataSuccess;
  } else {
    ++store->nGetDataFailure;
    rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
    interest->diskData = NULL;
  }
  store->getDataCb(npkt, store->getDataCtx);
}

__attribute__((nonnull)) static void
GetData_End(BdevRequest* breq, int res);

__attribute__((nonnull)) static inline void
GetData_Begin(DiskStore* store, DiskStoreRequest* req, Packet* npkt, uint64_t slotID)
{
  ++store->nGetDataBegin;
  N_LOGD("GetData begin slot=%" PRIu64 " npkt=%p", slotID, npkt);
  uint64_t blockOffset = slotID * store->nBlocksPerSlot;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  req->breq.pkt = Packet_ToMbuf(interest->diskData);
  req->breq.sp = RTE_PTR_ADD(rte_mbuf_to_priv(req->breq.pkt), sizeof(DiskStoreSlimRequest));
  req->breq.cb = GetData_End;
  Bdev_ReadPacket(&store->bdev, store->ch, blockOffset, &req->breq);
}

__attribute__((nonnull)) static void
DiskStore_ProcessQueue(DiskStore* store, DiskStoreRequest* head, struct rte_mbuf* dataPkt, int res)
{
  uint64_t slotID = head->s.slotID;

  while (head->s.next != NULL) {
    head->npkt = head->s.next;
    head->s.next = DiskStoreSlimRequest_FromPacket(head->npkt)->next;
    if (Packet_GetType(head->npkt) == PktData) {
      // process first queued PutData; later requests must wait
      PutData_Begin(store, head, head->npkt, slotID);
      return;
    }

    ++store->nGetDataReuse;
    if (unlikely(res != 0)) {
      // current request failed, subsequent GetData requests will fail too
      GetData_Finish(store, head->npkt, res);
      continue;
    }

    PInterest* interest = Packet_GetInterestHdr(head->npkt);
    struct rte_mbuf* dataBuf = Packet_ToMbuf(interest->diskData);
    dataBuf->data_off = 0;
    char* room = rte_pktmbuf_append(dataBuf, dataPkt->pkt_len);
    if (unlikely(room == NULL)) {
      // requester made a mistake and provided insufficient room
      GetData_Finish(store, head->npkt, ENOBUFS);
      continue;
    }

    // copy the packet payload from current request
    rte_memcpy(room, rte_pktmbuf_mtod(dataPkt, void*), dataPkt->pkt_len);
    Mbuf_SetTimestamp(dataBuf, Mbuf_GetTimestamp(dataPkt));
    bool ok = Packet_Parse(interest->diskData, ParseForFw);
    NDNDPDK_ASSERT(ok);
    NDNDPDK_ASSERT(Packet_GetType(interest->diskData) == PktData);

    N_LOGD("GetData success-reuse slot=%" PRIu64 " npkt=%p", slotID, head->npkt);
    GetData_Finish(store, head->npkt, 0);
  }

  // queue is empty, free the request index
  int32_t index = rte_hash_del_key(store->requestHt, &slotID);
  NDNDPDK_ASSERT(index >= 0);
  NDNDPDK_ASSERT(head == &store->requestArray[index]);
  head->s.slotID = 0; // let DiskStore_Process know this request index is unused
}

static void
PutData_End(BdevRequest* breq, int res)
{
  DiskStoreRequest* req = container_of(breq, DiskStoreRequest, breq);
  DiskStore* store = req->s.store;
  uint64_t slotID = req->s.slotID;
  Packet* npkt = req->npkt;

  if (likely(res == 0)) {
    N_LOGD("PutData success slot=%" PRIu64 " npkt=%p", slotID, npkt);
  } else {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID, npkt, res);
  }

  DiskStore_ProcessQueue(store, req, req->pkt, res);
  PutData_Finish(store, npkt, res);
}

static void
GetData_End(BdevRequest* breq, int res)
{
  DiskStoreRequest* req = container_of(breq, DiskStoreRequest, breq);
  DiskStore* store = req->s.store;
  uint64_t slotID = req->s.slotID;
  Packet* npkt = req->npkt;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);

  if (unlikely(res != 0)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID, npkt, res);
    goto FINISH;
  }

  Mbuf_SetTimestamp(dataPkt, rte_get_tsc_cycles());
  if (unlikely(!Packet_Parse(interest->diskData, ParseForFw)) ||
      unlikely(Packet_GetType(interest->diskData) != PktData)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("not-Data"), slotID, npkt);
    res = ENOEXEC;
    goto FINISH;
  }

  N_LOGD("GetData success slot=%" PRIu64 " npkt=%p", slotID, npkt);
FINISH:
  DiskStore_ProcessQueue(store, req, dataPkt, res);
  GetData_Finish(store, npkt, res);
}

static const struct
{
  const char* verb;
  void (*begin)(DiskStore* store, DiskStoreRequest* req, Packet* npkt, uint64_t slotID);
  void (*finish)(DiskStore* store, Packet* npkt, int res);
} DiskStoreOps[] = {
  [PktData] = {
    .verb = "PutData",
    .begin = PutData_Begin,
    .finish = PutData_Finish,
  },
  [PktInterest] = {
    .verb = "GetData",
    .begin = GetData_Begin,
    .finish = GetData_Finish,
  },
};

__attribute__((nonnull)) static void
DiskStore_Process(void* ctx)
{
  Packet* npkt = ctx;
  DiskStoreSlimRequest* sr = DiskStoreSlimRequest_FromPacket(npkt);
  DiskStore* store = sr->store;
  uint64_t slotID = sr->slotID;

  if (unlikely(store->ch == NULL)) {
    store->ch = spdk_bdev_get_io_channel(store->bdev.desc);
    if (unlikely(store->ch == NULL)) {
      rte_panic("spdk_bdev_get_io_channel error");
    }
  }

  NDNDPDK_ASSERT(store->ch != NULL);
  int32_t index = rte_hash_add_key(store->requestHt, &slotID);
  if (unlikely(index < 0)) {
    N_LOGW("%s rte_hash_add_key error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO,
           DiskStoreOps[Packet_GetType(npkt)].verb, slotID, npkt, (int)index);
    DiskStoreOps[Packet_GetType(npkt)].finish(store, npkt, ENOMEM);
  }
  DiskStoreRequest* req = &store->requestArray[index];

  // no other ongoing requests on the same slotID, begin the request right away
  if (likely(req->s.slotID == 0)) {
    req->s = *sr;
    req->npkt = npkt;
    DiskStoreOps[Packet_GetType(npkt)].begin(store, req, npkt, slotID);
    return;
  }

  // queue this request after other requests on the same slotID
  DiskStoreSlimRequest* tail = &req->s;
  int queueLen = 1;
  while (tail->next != NULL) {
    ++queueLen;
    tail = DiskStoreSlimRequest_FromPacket(tail->next);
  }
  tail->next = npkt;
  N_LOGD("%s queued queue-len=%d slot=%" PRIu64 " npkt=%p", DiskStoreOps[Packet_GetType(npkt)].verb,
         queueLen, slotID, npkt);
}

__attribute__((nonnull)) static inline void
DiskStore_Post(DiskStore* store, uint64_t slotID, Packet* npkt, DiskStoreSlimRequest* sr)
{
  N_LOGD("%s request slot=%" PRIu64 " npkt=%p", DiskStoreOps[Packet_GetType(npkt)].verb, slotID,
         npkt);
  *sr = (DiskStoreSlimRequest){
    .store = store,
    .slotID = slotID,
  };

  int res = spdk_thread_send_msg(store->th, DiskStore_Process, npkt);
  if (unlikely(res != 0)) {
    N_LOGW("%s spdk_thread_send_msg error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO,
           DiskStoreOps[Packet_GetType(npkt)].verb, slotID, npkt, res);
    DiskStoreOps[Packet_GetType(npkt)].finish(store, npkt, ENOSR);
  }
}

void
DiskStore_PutData(DiskStore* store, uint64_t slotID, Packet* npkt, BdevStoredPacket* sp)
{
  NDNDPDK_ASSERT(slotID > 0);

  uint32_t blockCount = BdevStoredPacket_ComputeBlockCount(sp);
  if (unlikely(blockCount > store->nBlocksPerSlot)) {
    N_LOGW("PutData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("packet-too-long"), slotID, npkt);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  PData* data = Packet_GetDataHdr(npkt);
  DiskStoreSlimRequest* sr = DiskStoreSlimRequest_FromData(data);
  DiskStore_Post(store, slotID, npkt, sr);
}

void
DiskStore_GetData(DiskStore* store, uint64_t slotID, Packet* npkt, struct rte_mbuf* dataBuf,
                  BdevStoredPacket* sp)
{
  NDNDPDK_ASSERT(slotID > 0);
  PInterest* interest = Packet_GetInterestHdr(npkt);
  interest->diskSlot = slotID;
  interest->diskData = Packet_FromMbuf(dataBuf);

  static_assert(sizeof(DiskStoreSlimRequest) + sizeof(BdevStoredPacket) <= sizeof(PacketPriv), "");
  DiskStoreSlimRequest* sr = rte_mbuf_to_priv(dataBuf);
  BdevStoredPacket_Copy(RTE_PTR_ADD(sr, sizeof(*sr)), sp);
  DiskStore_Post(store, slotID, npkt, sr);
}
