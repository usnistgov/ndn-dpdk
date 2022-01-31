#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

/** @brief Parameters related to GetData, stored in mbuf private area. */
typedef struct GetDataRequest
{
  DiskStore* store;
  struct rte_ring* reply;
} GetDataRequest;
static_assert(sizeof(GetDataRequest) <= sizeof(PacketPriv), "");

__attribute__((nonnull(2))) static void
GetData_Fail(struct rte_ring* reply, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  if (likely(interest->diskData != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
    interest->diskData = NULL;
  }

  if (likely(reply != NULL)) {
    if (likely(rte_ring_enqueue(reply, npkt) == 0)) {
      return;
    }
    N_LOGW("GetData error reply=%p npkt=%p" N_LOG_ERROR("enqueue"), reply, npkt);
  }
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

__attribute__((nonnull)) static void
GetData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  struct rte_ring* reply = ((GetDataRequest*)rte_mbuf_to_priv(dataPkt))->reply;

  if (unlikely(!success)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("io-err"), slotID, npkt);
    GetData_Fail(reply, npkt);
  } else {
    Mbuf_SetTimestamp(dataPkt, rte_get_tsc_cycles());
    if (unlikely(!Packet_Parse(interest->diskData)) ||
        unlikely(Packet_GetType(interest->diskData) != PktData)) {
      N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("not-Data"), slotID, npkt);
      GetData_Fail(reply, npkt);
    } else if (unlikely(rte_ring_enqueue(reply, npkt) != 0)) {
      N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("enqueue"), slotID, npkt);
      GetData_Fail(NULL, npkt);
    }
  }

  spdk_bdev_free_io(io);
}

__attribute__((nonnull)) static void
GetData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  GetDataRequest* req = (GetDataRequest*)rte_mbuf_to_priv(dataPkt);
  DiskStore* store = req->store;

  uint64_t blockOffset = DiskStore_ComputeBlockOffset_(store, slotID);

  int res = SpdkBdev_ReadPacket(store->bdev, store->ch, dataPkt, blockOffset, store->nBlocksPerSlot,
                                store->blockSize, GetData_End, (uintptr_t)npkt);
  if (unlikely(res != 0)) {
    N_LOGW("GetData read error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID, npkt, res);
    GetData_Fail(req->reply, npkt);
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
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("resize"), slotID, npkt);
    GetData_Fail(reply, npkt);
    return;
  }

  GetDataRequest* req = (GetDataRequest*)rte_mbuf_to_priv(dataBuf);
  req->store = store;
  req->reply = reply;

  spdk_thread_send_msg(store->th, GetData_Begin, npkt);
}
