#define _POSIX_C_SOURCE 200809L
#include "server.h"
#include "../core/logger.h"
#include "naming.h"
#include <fcntl.h>
#include <unistd.h>

N_LOG_INIT(FileServer);

typedef struct FilePayloadPriv
{
  struct rte_mbuf* interest;
  struct iovec iov;
  uint64_t segment;
  int prefix;
  int fd;
} FilePayloadPriv;
static_assert(sizeof(FilePayloadPriv) <= sizeof(PacketPriv), "");

__attribute__((nonnull)) static inline bool
FileServer_ProcessInterest(FileServer* p, struct rte_mbuf* interest, struct rte_mbuf* payload)
{
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  LName name = PName_ToLName(&pi->name);

  FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
  priv->prefix = LNamePrefixFilter_Find(name, FileServerMaxMounts, p->prefixL, p->prefixV);
  if (unlikely(priv->prefix < 0)) {
    N_LOGD("I drop=no-prefix");
    return false;
  }

  char* filename = rte_pktmbuf_mtod_offset(interest, char*, interest->data_len);
  uint16_t suffixOff =
    FileServer_NameToPath(name, p->prefixL[priv->prefix], filename, rte_pktmbuf_tailroom(interest));
  if (unlikely(suffixOff == UINT16_MAX)) {
    N_LOGD("I drop=bad-name");
    return false;
  }
  FileServerSuffix suffix = FileServer_ParseSuffix(name, suffixOff);
  if (unlikely(!suffix.ok)) {
    N_LOGD("I drop=bad-suffix");
    return false;
  }

  if (unlikely(!suffix.hasSegment)) {
    // "32=ls" and "32=metadata" not implemented
    N_LOGD("I drop=keyword-not-implemented");
    return false;
  }

  priv->fd = openat(p->dfd[priv->prefix], filename, O_RDONLY);
  if (unlikely(priv->fd < 0)) {
    N_LOGD("I drop=openat-error errno=%d", errno);
    return false;
  }

  struct io_uring_sqe* sqe = io_uring_get_sqe(&p->uring);
  if (unlikely(sqe == NULL)) {
    N_LOGD("I drop=no-sqe");
    close(priv->fd);
    return false;
  }

  priv->interest = interest;
  priv->segment = suffix.segment;

  payload->data_off = p->payloadHeadroom;
  priv->iov.iov_base = rte_pktmbuf_mtod(payload, uint8_t*);
  priv->iov.iov_len = p->segmentLen + 1;

  io_uring_prep_readv(sqe, priv->fd, &priv->iov, 1, suffix.segment * p->segmentLen);
  io_uring_sqe_set_data(sqe, payload);
  return true;
}

__attribute__((nonnull)) static inline uint32_t
FileServer_RxBurst(FileServer* p)
{
  TscTime now = rte_get_tsc_cycles();
  struct rte_mbuf* interest[MaxBurstSize];
  struct rte_mbuf* payload[MaxBurstSize * 2];

  PktQueuePopResult pop = PktQueue_Pop(&p->rxQueue, interest, MaxBurstSize, now);
  if (unlikely(pop.count == 0)) {
    return 0;
  }

  int res = rte_pktmbuf_alloc_bulk(p->payloadMp, payload, pop.count);
  if (unlikely(res != 0)) {
    rte_pktmbuf_free_bulk(interest, pop.count);
    return pop.count;
  }

  uint32_t nPayload = 0, discardPos = pop.count;
  for (uint32_t i = 0; i < pop.count; ++i) {
    if (likely(FileServer_ProcessInterest(p, interest[i], payload[nPayload]))) {
      ++nPayload;
    } else {
      payload[discardPos++] = interest[i];
    }
  }

  if (likely(nPayload > 0)) {
    res = io_uring_submit(&p->uring);
    if (unlikely(res < 0)) {
      N_LOGW("io_uring_submit errno=%d", -res);
    }
  }
  if (unlikely(nPayload != discardPos)) {
    rte_pktmbuf_free_bulk(&payload[nPayload], discardPos - nPayload);
  }
  return pop.count;
}

__attribute__((nonnull)) static inline bool
FileServer_ProcessCqe(FileServer* p, struct io_uring_cqe* cqe, TscTime now)
{
  if (unlikely(cqe->res < 0)) {
    N_LOGD("C drop=cqe-error errno=%d", -cqe->res);
    return false;
  }

  struct rte_mbuf* payload = io_uring_cqe_get_data(cqe);
  rte_pktmbuf_append(payload, RTE_MIN((uint16_t)cqe->res, p->segmentLen));

  FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
  uint64_t segment = priv->segment;
  Packet* interest = Packet_FromMbuf(priv->interest);
  PInterest* pi = Packet_GetInterestHdr(interest);
  LName name = PName_ToLName(&pi->name);

  LName finalBlock = { 0 };
  uint8_t finalBlockV[10];
  if (unlikely((uint16_t)cqe->res <= p->segmentLen)) {
    finalBlockV[0] = TtSegmentNameComponent;
    finalBlockV[1] = Nni_Encode(&finalBlockV[2], segment);
    finalBlock.length = 2 + finalBlockV[1];
    finalBlock.value = finalBlockV;
  }
  MetaInfoBuffer meta;
  DataEnc_PrepareMetaInfo(&meta, ContentBlob, 300000, finalBlock);

  NULLize(priv); // overwritten by DataEnc
  Packet* data = DataEnc_EncodePayload(name, &meta, payload);
  if (unlikely(data == NULL)) {
    N_LOGD("C drop=dataenc-error");
    return false;
  }

  Mbuf_SetTimestamp(payload, Mbuf_GetTimestamp(Packet_ToMbuf(interest)));
  LpL3* dataLpL3 = Packet_GetLpL3Hdr(data);
  LpL3* interestLpL3 = Packet_GetLpL3Hdr(interest);
  dataLpL3->pitToken = interestLpL3->pitToken;
  dataLpL3->congMark = interestLpL3->congMark;
  return true;
}

__attribute__((nonnull)) static inline uint32_t
FileServer_TxBurst(FileServer* p)
{
  TscTime now = rte_get_tsc_cycles();

  struct io_uring_cqe* cqe[MaxBurstSize];
  uint32_t nCqe = io_uring_peek_batch_cqe(&p->uring, cqe, RTE_DIM(cqe));

  struct rte_mbuf* data[2 * MaxBurstSize];
  struct rte_mbuf* discard[2 * MaxBurstSize];
  uint32_t nData = 0, nDiscard = 0;
  for (uint32_t i = 0; i < nCqe; ++i) {
    struct rte_mbuf* payload = io_uring_cqe_get_data(cqe[i]);
    FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
    close(priv->fd);
    discard[nDiscard++] = priv->interest;
    NULLize(priv); // overwritten by DataEnc

    if (likely(FileServer_ProcessCqe(p, cqe[i], now))) {
      data[nData++] = payload;
    } else {
      discard[nDiscard++] = payload;
    }
    io_uring_cqe_seen(&p->uring, cqe[i]);
  }

  Face_TxBurst(p->face, (Packet**)data, nData);
  rte_pktmbuf_free_bulk(discard, nDiscard);
  return nCqe;
}

int
FileServer_Run(FileServer* p)
{
  int res = io_uring_queue_init(p->uringCapacity, &p->uring, 0);
  if (res < 0) {
    N_LOGE("io_uring_queue_init errno=%d", -res);
    return 1;
  }

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(p->ctrl, nProcessed)) {
    nProcessed += FileServer_RxBurst(p);
    nProcessed += FileServer_TxBurst(p);
  }

  io_uring_queue_exit(&p->uring);
  return 0;
}
