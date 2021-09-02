#include "server.h"
#include "../core/logger.h"
#include "naming.h"

N_LOG_INIT(FileServer);

typedef struct FilePayloadPriv
{
  FileServerFd* fd;
  struct rte_mbuf* interest;
  uint64_t segment;
  struct iovec iov[1];
} FilePayloadPriv;
static_assert(sizeof(FilePayloadPriv) <= sizeof(PacketPriv), "");

__attribute__((nonnull)) static inline bool
FileServer_ProcessInterest(FileServer* p, struct rte_mbuf* interest, struct rte_mbuf* payload)
{
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  FileServerRequestName rn;
  if (unlikely(!FileServer_ParseRequest(&rn, &pi->name))) {
    N_LOGD("I drop=bad-name");
    return false;
  }

  if (unlikely(!rn.hasSegment)) {
    // "32=ls" and "32=metadata" not implemented
    N_LOGD("I drop=keyword-not-implemented");
    return false;
  }

  FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
  priv->fd = FileServer_FdOpen(p, &pi->name);
  if (unlikely(priv->fd == NULL)) {
    N_LOGD("I drop=no-fd");
    return false;
  }
  if (unlikely(priv->fd == FileServer_NotFound)) {
    N_LOGD("I drop=file-not-found");
    return false;
  }
  if (unlikely(rn.segment > priv->fd->lastSeg)) {
    N_LOGD("I drop=segment-out-of-range segment=%" PRIu64 " lastseg=%" PRIu64, rn.segment,
           priv->fd->lastSeg);
    goto UNREF;
  }

  struct io_uring_sqe* sqe = io_uring_get_sqe(&p->uring);
  if (unlikely(sqe == NULL)) {
    N_LOGE("I" N_LOG_ERROR("no-sqe"));
    goto UNREF;
  }

  priv->interest = interest;

  payload->data_off = p->payloadHeadroom;
  priv->iov[0].iov_base = rte_pktmbuf_mtod(payload, uint8_t*);
  priv->iov[0].iov_len = p->segmentLen;

  io_uring_prep_readv(sqe, priv->fd->fd, priv->iov, 1, rn.segment * p->segmentLen);
  io_uring_sqe_set_data(sqe, payload);
  return true;

UNREF:
  FileServer_FdUnref(p, priv->fd);
  return false;
}

__attribute__((nonnull)) static inline uint32_t
FileServer_RxBurst(FileServer* p)
{
  TscTime now = rte_get_tsc_cycles();
  struct rte_mbuf* interest[MaxBurstSize];
  struct rte_mbuf* payload[MaxBurstSize * 2];

  PktQueuePopResult pop = PktQueue_Pop(&p->rxQueue, interest, MaxBurstSize, now);
  if (unlikely(pop.count == 0)) {
    return pop.count;
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
  rte_pktmbuf_append(payload, (uint16_t)cqe->res);

  FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
  Packet* interest = Packet_FromMbuf(priv->interest);
  PInterest* pi = Packet_GetInterestHdr(interest);
  LName name = PName_ToLName(&pi->name);

  Packet* data = DataEnc_EncodePayload(name, &priv->fd->meta, payload);
  NULLize(priv); // overwritten by DataEnc
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
  uint32_t nData = 0, discardInterest = MaxBurstSize, discardPayload = MaxBurstSize;
  for (uint32_t i = 0; i < nCqe; ++i) {
    struct rte_mbuf* payload = io_uring_cqe_get_data(cqe[i]);
    FilePayloadPriv* priv = rte_mbuf_to_priv(payload);
    FileServerFd* fd = priv->fd;
    discard[discardInterest++] = priv->interest;
    NULLize(priv); // overwritten by DataEnc
    if (likely(FileServer_ProcessCqe(p, cqe[i], now))) {
      data[nData++] = payload;
    } else {
      discard[--discardPayload] = payload;
    }
    FileServer_FdUnref(p, fd);
    NULLize(fd);
    io_uring_cqe_seen(&p->uring, cqe[i]);
  }

  Face_TxBurst(p->face, (Packet**)data, nData);
  rte_pktmbuf_free_bulk(&discard[discardPayload], discardInterest - discardPayload);
  return nCqe;
}

int
FileServer_Run(FileServer* p)
{
  struct io_uring_params uringParams = { 0 };
  int res = io_uring_queue_init_params(p->uringCapacity, &p->uring, &uringParams);
  if (res < 0) {
    N_LOGE("uring init errno=%d", -res);
    return 1;
  }
  N_LOGI("uring init sqe=%" PRIu32 " cqe=%" PRIu32 " features=%" PRIx32, uringParams.sq_entries,
         uringParams.cq_entries, uringParams.features);
  TAILQ_INIT(&p->fdQ);

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(p->ctrl, nProcessed)) {
    nProcessed += FileServer_RxBurst(p);
    nProcessed += FileServer_TxBurst(p);
  }

  io_uring_queue_exit(&p->uring);
  FileServer_FdClear(p);
  return 0;
}
