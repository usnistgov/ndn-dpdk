#include "store.h"

#include "../core/logger.h"

N_LOG_INIT(DiskStore);

/** @brief Parameters related to GetData, stored in mbuf private area. */
typedef struct GetDataRequest
{
  DiskStore* store;
} GetDataRequest;
static_assert(sizeof(GetDataRequest) <= sizeof(PacketPriv), "");

__attribute__((nonnull)) static void
GetData_Fail(DiskStore* store, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  if (likely(interest->diskData != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
    interest->diskData = NULL;
  }

  store->getDataCb(npkt, store->getDataCtx);
}

__attribute__((nonnull)) static void
GetData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  DiskStore* store = ((GetDataRequest*)rte_mbuf_to_priv(dataPkt))->store;

  if (unlikely(!success)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("io-err"), slotID, npkt);
    goto FAIL;
  }

  Mbuf_SetTimestamp(dataPkt, rte_get_tsc_cycles());
  if (unlikely(!Packet_Parse(interest->diskData)) ||
      unlikely(Packet_GetType(interest->diskData) != PktData)) {
    N_LOGW("GetData error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR("not-Data"), slotID, npkt);
    goto FAIL;
  }

  store->getDataCb(npkt, store->getDataCtx);
  goto FREE;

FAIL:
  GetData_Fail(store, npkt);
FREE:
  spdk_bdev_free_io(io);
}

__attribute__((nonnull)) static void
GetData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotID = interest->diskSlot;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  DiskStore* store = ((GetDataRequest*)rte_mbuf_to_priv(dataPkt))->store;

  if (unlikely(store->ch == NULL)) {
    store->ch = spdk_bdev_get_io_channel(store->bdev);
    if (unlikely(store->ch == NULL)) {
      N_LOGW("GetData no I/O channel" N_LOG_ERROR("spdk_bdev_get_io_channel"));
      GetData_Fail(store, npkt);
      return;
    }
  }

  uint64_t blockOffset = DiskStore_ComputeBlockOffset_(store, slotID);
  int res = SpdkBdev_ReadPacket(store->bdev, store->ch, dataPkt, blockOffset, store->nBlocksPerSlot,
                                store->blockSize, GetData_End, (uintptr_t)npkt);
  if (unlikely(res != 0)) {
    N_LOGW("GetData read error slot=%" PRIu64 " npkt=%p" N_LOG_ERROR_ERRNO, slotID, npkt, res);
    GetData_Fail(store, npkt);
  }
}

void
DiskStore_GetData(DiskStore* store, uint64_t slotID, Packet* npkt, struct rte_mbuf* dataBuf)
{
  NDNDPDK_ASSERT(slotID > 0);
  PInterest* interest = Packet_GetInterestHdr(npkt);
  interest->diskSlot = slotID;
  interest->diskData = Packet_FromMbuf(dataBuf);

  GetDataRequest* req = (GetDataRequest*)rte_mbuf_to_priv(dataBuf);
  req->store = store;

  spdk_thread_send_msg(store->th, GetData_Begin, npkt);
}
