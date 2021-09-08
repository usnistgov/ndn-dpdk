#include "server.h"

#include "../core/logger.h"
#include "fd.h"
#include "op.h"

N_LOG_INIT(FileServer);

enum
{
  MaxBurstIovecs = MaxBurstSize * FileServerMaxIovecs,
};

typedef struct TxBurstCtx
{
  TscTime now;
  uint16_t nData; ///< data[nData] are Data packets to be transmitted
  /// discard[discardPayloadIndex : MaxBurstIovecs] are payload mbufs to be freed
  uint16_t discardPayloadIndex;
  /// discard[MaxBurstIovecs : discardInterestIndex] are Interest mbufs to be freed
  uint16_t discardInterestIndex;
  struct io_uring_cqe* cqe[MaxBurstSize];
  Packet* data[MaxBurstIovecs];
  struct rte_mbuf* discard[MaxBurstIovecs * 2];
} TxBurstCtx;
static_assert(RTE_DIM(((TxBurstCtx*)NULL)->discard) <= UINT16_MAX, "");

__attribute__((nonnull)) static inline void
FileServerTx_ProcessCqe(FileServer* p, TxBurstCtx* ctx, uint32_t i)
{
  struct io_uring_cqe* cqe = ctx->cqe[i];
  FileServerOp* op = io_uring_cqe_get_data(cqe);
  FileServerFd* fd = op->fd;
  uint32_t nIov = op->nIov;

  if (unlikely(cqe->res < 0)) {
    N_LOGD("CQE fd=%d nIov=%" PRIu32 " drop=cqe-error" N_LOG_ERROR("errno=%d"), fd->fd, nIov,
           -cqe->res);
    for (uint32_t i = 0; i < op->nIov; ++i) {
      struct rte_mbuf* payload = NULL;
      struct rte_mbuf* interest = NULL;
      FileServerOpMbufs_Get(&op->mbufs, i, &payload, &interest);
      ctx->discard[--ctx->discardPayloadIndex] = payload;
      ctx->discard[ctx->discardInterestIndex++] = interest;
    }
    goto FINISH;
  }

  N_LOGV("CQE fd=%d nIov=%" PRIu32 " res=%" PRId32, fd->fd, nIov, (int32_t)cqe->res);
  FileServerOpMbufs mbufs;
  FileServerOpMbufs_Copy(&mbufs, &op->mbufs, nIov);
  NULLize(op); // overwritten by DataEnc

  uint32_t totalLen = cqe->res;
  for (uint32_t i = 0; i < nIov; ++i) {
    struct rte_mbuf* payload = NULL;
    struct rte_mbuf* interestPkt = NULL;
    FileServerOpMbufs_Get(&mbufs, i, &payload, &interestPkt);

    Packet* interest = Packet_FromMbuf(interestPkt);
    PInterest* pi = Packet_GetInterestHdr(interest);
    LName name = PName_ToLName(&pi->name);
    ctx->discard[ctx->discardInterestIndex++] = interestPkt;

    uint16_t segmentLen = RTE_MIN(p->segmentLen, totalLen);
    totalLen -= segmentLen;
    rte_pktmbuf_append(payload, segmentLen);

    Packet* data = DataEnc_EncodePayload(name, (LName){ 0 }, &fd->meta, payload);
    if (unlikely(data == NULL)) {
      N_LOGD("CQE drop=dataenc-error");
      ctx->discard[--ctx->discardPayloadIndex] = payload;
      continue;
    }

    Mbuf_SetTimestamp(payload, ctx->now);
    *Packet_GetLpL3Hdr(data) = *Packet_GetLpL3Hdr(interest);
    ctx->data[ctx->nData++] = data;
  }

FINISH:
  FileServerFd_Unref(p, fd);
  io_uring_cqe_seen(&p->uring, cqe);
}

uint32_t
FileServer_TxBurst(FileServer* p)
{
  TxBurstCtx ctx;
  ctx.now = rte_get_tsc_cycles();
  ctx.nData = 0;
  ctx.discardPayloadIndex = MaxBurstIovecs;
  ctx.discardInterestIndex = MaxBurstIovecs;

  uint32_t nCqe = io_uring_peek_batch_cqe(&p->uring, ctx.cqe, RTE_DIM(ctx.cqe));
  for (uint32_t i = 0; i < nCqe; ++i) {
    FileServerTx_ProcessCqe(p, &ctx, i);
  }

  Face_TxBurst(p->face, ctx.data, ctx.nData);
  rte_pktmbuf_free_bulk(&ctx.discard[ctx.discardPayloadIndex],
                        ctx.discardInterestIndex - ctx.discardPayloadIndex);
  return nCqe;
}
