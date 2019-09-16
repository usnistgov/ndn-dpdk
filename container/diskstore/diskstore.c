#include "diskstore.h"

#include "../../core/logger.h"

INIT_ZF_LOG(DiskStore);

static uint64_t
DiskStore_ComputeBlockOffset(DiskStore* store, uint64_t slotId)
{
  return slotId * store->nBlocksPerSlot;
}

static uint64_t
DiskStore_ComputeBlockCount(DiskStore* store, Packet* npkt)
{
  uint64_t pktLen = Packet_ToMbuf(npkt)->pkt_len;
  return pktLen / DISK_STORE_BLOCK_SIZE +
         (int)(pktLen % DISK_STORE_BLOCK_SIZE > 0);
}

/** \brief Parameters related to PutData, stored over PData.digest field.
 */
typedef struct DiskStore_PutDataRequest
{
  DiskStore* store;
  uint64_t slotId;
} DiskStore_PutDataRequest;
static_assert(sizeof(DiskStore_PutDataRequest) <=
                sizeof(((PData*)(NULL))->digest),
              "");

static void
DiskStore_PutData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PData* data = Packet_GetDataHdr(npkt);
  DiskStore_PutDataRequest* req = (DiskStore_PutDataRequest*)&data->digest[0];
  DiskStore* store = req->store;
  uint64_t slotId = req->slotId;

  if (unlikely(!success)) {
    ZF_LOGW("PutData_End(%" PRIu64 ", %p): fail=io-err", slotId, npkt);
  }

  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  spdk_bdev_free_io(io);
}

static void
DiskStore_PutData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PData* data = Packet_GetDataHdr(npkt);
  DiskStore_PutDataRequest* req = (DiskStore_PutDataRequest*)&data->digest[0];
  DiskStore* store = req->store;
  uint64_t slotId = req->slotId;

  uint64_t blockOffset = DiskStore_ComputeBlockOffset(store, slotId);
  uint64_t blockCount = DiskStore_ComputeBlockCount(store, npkt);
  int res = SpdkBdev_WritePacket(store->bdev,
                                 store->ch,
                                 Packet_ToMbuf(npkt),
                                 blockOffset,
                                 blockCount,
                                 store->blockSize,
                                 DiskStore_PutData_End,
                                 npkt);
  if (unlikely(res != 0)) {
    ZF_LOGW(
      "PutData_Begin(%" PRIu64 ", %p): fail=write(%d)", slotId, npkt, res);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
  }
}

void
DiskStore_PutData(DiskStore* store, uint64_t slotId, Packet* npkt)
{
  assert(slotId > 0);
  uint64_t blockCount = DiskStore_ComputeBlockCount(store, npkt);
  if (unlikely(blockCount > store->nBlocksPerSlot)) {
    ZF_LOGW("PutData(%" PRIu64 ", %p): fail=packet-too-long", slotId, npkt);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  PData* data = Packet_GetDataHdr(npkt);
  data->hasDigest = false;
  DiskStore_PutDataRequest* req = (DiskStore_PutDataRequest*)&data->digest[0];
  req->store = store;
  req->slotId = slotId;
  spdk_thread_send_msg(store->th, DiskStore_PutData_Begin, npkt);
}

/** \brief Parameters related to GetData, stored in mbuf private area.
 */
typedef struct DiskStore_GetDataRequest
{
  DiskStore* store;
  struct rte_ring* reply;
} DiskStore_GetDataRequest;

static void
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
    ZF_LOGW("GetData_Fail(%p, %p): fail=enqueue", reply, npkt);
  }
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

static void
DiskStore_GetData_End(struct spdk_bdev_io* io, bool success, void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotId = interest->diskSlotId;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  struct rte_ring* reply =
    ((DiskStore_GetDataRequest*)rte_mbuf_to_priv_(dataPkt))->reply;

  if (unlikely(!success)) {
    ZF_LOGW("GetData_End(%" PRIu64 ", %p): fail=io-err", slotId, npkt);
    DiskStore_GetData_Fail(reply, npkt);
  } else {
    dataPkt->timestamp = rte_get_tsc_cycles();
    NdnError err = Packet_ParseL3(interest->diskData, NULL);
    if (err != NdnError_OK ||
        Packet_GetL3PktType(interest->diskData) != L3PktType_Data) {
      ZF_LOGW("GetData_End(%" PRIu64 ", %p): fail=not-Data", slotId, npkt);
      DiskStore_GetData_Fail(reply, npkt);
    } else {
      if (unlikely(rte_ring_enqueue(reply, npkt) != 0)) {
        ZF_LOGW("GetData_End(%" PRIu64 ", %p): fail=enqueue", slotId, npkt);
        DiskStore_GetData_Fail(NULL, npkt);
      }
    }
  }

  spdk_bdev_free_io(io);
}

static void
DiskStore_GetData_Begin(void* npkt0)
{
  Packet* npkt = (Packet*)npkt0;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t slotId = interest->diskSlotId;
  struct rte_mbuf* dataPkt = Packet_ToMbuf(interest->diskData);
  DiskStore_GetDataRequest* req =
    (DiskStore_GetDataRequest*)rte_mbuf_to_priv_(dataPkt);
  DiskStore* store = req->store;

  uint64_t blockOffset = DiskStore_ComputeBlockOffset(store, slotId);

  int res = SpdkBdev_ReadPacket(store->bdev,
                                store->ch,
                                dataPkt,
                                blockOffset,
                                store->nBlocksPerSlot,
                                store->blockSize,
                                DiskStore_GetData_End,
                                npkt);
  if (unlikely(res != 0)) {
    ZF_LOGW("GetData_Begin(%" PRIu64 ", %p): fail=read(%d)", slotId, npkt, res);
    DiskStore_GetData_Fail(req->reply, npkt);
  }
}

void
DiskStore_GetData(DiskStore* store,
                  uint64_t slotId,
                  uint16_t dataLen,
                  Packet* npkt,
                  struct rte_ring* reply)
{
  assert(slotId > 0);
  PInterest* interest = Packet_GetInterestHdr(npkt);
  assert(interest->diskSlotId ==
         0); // an Interest can go through DiskStore_GetData only once
  interest->diskSlotId = slotId;
  interest->diskData = NULL;

  // TODO allocate from a mempool in caller's NUMA socket
  struct rte_mbuf* dataPkt = rte_pktmbuf_alloc(store->mp);
  if (unlikely(dataPkt == NULL)) {
    ZF_LOGW("GetData(%" PRIu64 ", %p): fail=alloc-err", slotId, npkt);
    DiskStore_GetData_Fail(reply, npkt);
    return;
  }
  interest->diskData = Packet_FromMbuf(dataPkt);

  if (unlikely(rte_pktmbuf_append(dataPkt, dataLen) == NULL)) {
    ZF_LOGW("GetData(%" PRIu64 ", %p): fail=resize-err", slotId, npkt);
    DiskStore_GetData_Fail(reply, npkt);
    return;
  }

  assert(dataPkt->priv_size >= sizeof(DiskStore_GetDataRequest));
  DiskStore_GetDataRequest* req =
    (DiskStore_GetDataRequest*)rte_mbuf_to_priv_(dataPkt);
  req->store = store;
  req->reply = reply;

  spdk_thread_send_msg(store->th, DiskStore_GetData_Begin, npkt);
}
