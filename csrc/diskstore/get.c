#include "diskstore.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

/** @brief Parameters related to GetData, stored in mbuf private area. */
typedef struct DiskStore_GetDataRequest
{
  DiskStore* store;
  struct rte_ring* reply;
} DiskStore_GetDataRequest;
static_assert(sizeof(DiskStore_GetDataRequest) <= sizeof(PacketPriv), "");

__attribute__((nonnull(2))) static void
DiskStore_GetData_Fail(struct rte_ring* reply, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  if (likely(interest->diskData != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
    interest->diskData = NULL;
  }

  if (likely(reply != NULL) && likely(rte_ring_enqueue(reply, npkt) == 0)) {
    return;
  }
  if (reply != NULL) {
    N_LOGW("GetData_Fail reply=%p npkt=%p fail=enqueue", reply, npkt);
  }
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

__attribute__((nonnull)) static void
DiskStore_GetData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  struct rte_ring* reply = ((DiskStore_GetDataRequest*)rte_mbuf_to_priv(dataPkt))->reply;

  if (unlikely(!success)) {
    N_LOGW("GetData_End slot=%" PRIu64 " npkt=%p fail=io-err", slotID, npkt);
    DiskStore_GetData_Fail(reply, npkt);
  } else {
    Mbuf_SetTimestamp(dataPkt, rte_get_tsc_cycles());
    if (unlikely(!Packet_Parse(interest->diskData)) ||
        unlikely(Packet_GetType(interest->diskData) != PktData)) {
      N_LOGW("GetData_End slot=%" PRIu64 " npkt=%p fail=not-Data", slotID, npkt);
      DiskStore_GetData_Fail(reply, npkt);
    } else if (unlikely(rte_ring_enqueue(reply, npkt) != 0)) {
      N_LOGW("GetData_End slot=%" PRIu64 " npkt=%p fail=enqueue", slotID, npkt);
      DiskStore_GetData_Fail(NULL, npkt);
    }
  }

  spdk_bdev_free_io(io);
}

__attribute__((nonnull)) static void
DiskStore_GetData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  DiskStore_GetDataRequest* req = (DiskStore_GetDataRequest*)rte_mbuf_to_priv(dataPkt);
  DiskStore* store = req->store;

  uint64_t blockOffset = DiskStore_ComputeBlockOffset_(store, slotID);

  int res = SpdkBdev_ReadPacket(store->bdev, store->ch, dataPkt, blockOffset, store->nBlocksPerSlot,
                                store->blockSize, DiskStore_GetData_End, npkt);
  if (unlikely(res != 0)) {
    N_LOGW("GetData_Begin slot=%" PRIu64 " npkt=%p fail=read(%d)", slotID, npkt, res);
    DiskStore_GetData_Fail(req->reply, npkt);
  }
}

void
DiskStore_GetData(DiskStore* store, uint64_t slotID, uint16_t dataLen, Packet* npkt,
                  struct rte_mbuf* dataBuf, struct rte_ring* reply)
{
  NDNDPDK_ASSERT(slotID > 0);
  PInterest* interest = Packet_GetInterestHdr(npkt);
  interest->diskSlot = slotID;
  interest->diskData = Packet_FromMbuf(dataBuf);

  if (unlikely(rte_pktmbuf_append(dataBuf, dataLen) == NULL)) {
    N_LOGW("GetData slot=%" PRIu64 " npkt=%p fail=resize-err", slotID, npkt);
    DiskStore_GetData_Fail(reply, npkt);
    return;
  }

  DiskStore_GetDataRequest* req = (DiskStore_GetDataRequest*)rte_mbuf_to_priv(dataBuf);
  req->store = store;
  req->reply = reply;

  spdk_thread_send_msg(store->th, DiskStore_GetData_Begin, npkt);
}
