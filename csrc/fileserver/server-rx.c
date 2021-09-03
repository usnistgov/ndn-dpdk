#include "server.h"

#include "../core/logger.h"
#include "fd.h"
#include "naming.h"
#include "op.h"

N_LOG_INIT(FileServer);

typedef struct RxBurstCtx
{
  TscTime now;
  FileServerOp* op;
  uint8_t interestIndex; ///< interest[:interestIndex] are processed
  uint8_t interestCount; ///< interest[interestIndex:interestCount] are unprocessed
  uint8_t payloadIndex;  ///< payload[payloadIndex:] are unused
  uint8_t discardIndex;  ///< discard[MaxBurstSize:discardIndex] are dropped Interests
  bool hasSqe;
  char zeroizeEnd_[0];
  struct rte_mbuf* interest[MaxBurstSize];
  union
  {
    struct rte_mbuf* payload[MaxBurstSize];
    struct rte_mbuf* discard[2 * MaxBurstSize];
  };
} RxBurstCtx;
static_assert(RTE_DIM(((RxBurstCtx*)NULL)->discard) <= UINT8_MAX, "");

/**
 * @brief Handle SQE unavailable error.
 * @return false.
 * @post payload and Interest mbufs not yet in SQEs are in discard[payloadIndex:discardIndex].
 * @post FileServer_RxBurst packet processing loop is stopped
 */
__attribute__((nonnull)) static __rte_noinline bool
FileServerRx_NoSqe(RxBurstCtx* ctx)
{
  N_LOGW("SQE no-sqe" N_LOG_ERROR_BLANK);

  for (uint32_t i = 0; i < ctx->op->nIov; ++i) {
    struct rte_mbuf* payload = NULL;
    struct rte_mbuf* interest = NULL;
    FileServerOpMbufs_Get(&ctx->op->mbufs, i, &payload, &interest);
    ctx->payload[--ctx->payloadIndex] = payload;
    ctx->discard[ctx->discardIndex++] = interest;
  }
  ctx->op = NULL;

  rte_memcpy(&ctx->interest[ctx->interestIndex], &ctx->discard[ctx->discardIndex],
             sizeof(ctx->discard[0]) * (ctx->interestCount - ctx->interestIndex));
  ctx->interestIndex = ctx->interestCount;

  return false;
}

/**
 * @brief Queue readv SQE for current operation.
 * @return whether success.
 */
__attribute__((nonnull)) static inline bool
FileServerRx_Readv(FileServer* p, RxBurstCtx* ctx)
{
  struct io_uring_sqe* sqe = io_uring_get_sqe(&p->uring);
  if (unlikely(sqe == NULL)) {
    return FileServerRx_NoSqe(ctx);
  }
  ctx->hasSqe = true;

  FileServerOp* op = ctx->op;
  N_LOGV("SQE fd=%d segment=%" PRIu64 " nIov=%" PRIu32, op->fd->fd, op->segment, op->nIov);
  io_uring_prep_readv(sqe, op->fd->fd, op->iov, op->nIov, op->segment * p->segmentLen);
  io_uring_sqe_set_data(sqe, op);
  ctx->op = NULL;
  return true;
}

__attribute__((nonnull)) static inline void
FileServerRx_ProcessInterest(FileServer* p, RxBurstCtx* ctx)
{
  struct rte_mbuf* interest = ctx->interest[ctx->interestIndex];
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  FileServerRequestName rn;
  if (unlikely(!FileServer_ParseRequest(&rn, &pi->name))) {
    N_LOGD("I drop=bad-name");
    goto DROP;
  }

  if (unlikely(!rn.hasSegment)) {
    // "32=ls" and "32=metadata" not implemented
    N_LOGD("I drop=keyword-not-implemented");
    goto DROP;
  }

  LName prefix = FileServer_GetPrefix(&pi->name);
  struct rte_mbuf* payload = ctx->payload[ctx->payloadIndex];
  payload->data_off = p->payloadHeadroom;

  if (likely(ctx->op != NULL)) {
    if (likely(FileServerOp_IsContinuous(ctx->op, prefix, rn.segment))) {
      goto ADD_IOV;
    }
    if (unlikely(!FileServerRx_Readv(p, ctx))) {
      // not `goto DROP` because FileServerRx_Readv has dropped `interest`,
      // which was "unprocessed" until returning to RxBurst
      return;
    }
  }

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("I drop=no-fd");
    goto DROP;
  }
  if (unlikely(fd == FileServer_NotFound)) {
    N_LOGD("I drop=file-not-found");
    goto UNREF;
  }
  if (unlikely(rn.segment > fd->lastSeg)) {
    N_LOGD("I fd=%d drop=segment-out-of-range segment=%" PRIu64 " lastseg=%" PRIu64, fd->fd,
           rn.segment, fd->lastSeg);
    goto UNREF;
  }

  ctx->op = rte_mbuf_to_priv(payload);
  FileServerOp_Init(ctx->op, fd, prefix, rn.segment);

ADD_IOV:
  ctx->op->iov[ctx->op->nIov] = (struct iovec){
    .iov_base = rte_pktmbuf_mtod(payload, uint8_t*),
    .iov_len = p->segmentLen,
  };
  FileServerOpMbufs_Set(&ctx->op->mbufs, ctx->op->nIov, payload, interest);
  ++ctx->op->nIov;
  ++ctx->payloadIndex;

  if (unlikely(ctx->op->nIov == FileServerMaxIovecs)) {
    FileServerRx_Readv(p, ctx);
  }
  return;
UNREF:
  FileServerFd_Unref(p, fd);
DROP:
  ctx->discard[ctx->discardIndex++] = interest;
}

uint32_t
FileServer_RxBurst(FileServer* p)
{
  RxBurstCtx ctx;
  memset(&ctx, 0, offsetof(RxBurstCtx, zeroizeEnd_));
  ctx.now = rte_get_tsc_cycles();
  PktQueuePopResult pop = PktQueue_Pop(&p->rxQueue, ctx.interest, MaxBurstSize, ctx.now);
  if (unlikely(pop.count == 0)) {
    return pop.count;
  }

  ctx.interestCount = pop.count;
  ctx.payloadIndex = MaxBurstSize - ctx.interestCount;
  int res = rte_pktmbuf_alloc_bulk(p->payloadMp, &ctx.payload[ctx.payloadIndex], ctx.interestCount);
  if (unlikely(res != 0)) {
    rte_pktmbuf_free_bulk(ctx.interest, ctx.interestCount);
    return ctx.interestCount;
  }
  ctx.discardIndex = MaxBurstSize;

  for (; ctx.interestIndex < ctx.interestCount; ++ctx.interestIndex) {
    FileServerRx_ProcessInterest(p, &ctx);
    // upon failure, ctx.interestIndex is set to ctx.interestCount, stopping the loop
  }
  if (likely(ctx.op != NULL)) {
    FileServerRx_Readv(p, &ctx);
  }

  if (likely(ctx.hasSqe)) {
    res = io_uring_submit(&p->uring);
    if (unlikely(res < 0)) {
      N_LOGE("io_uring_submit errno=%d", -res);
    }
  }

  rte_pktmbuf_free_bulk(&ctx.discard[ctx.payloadIndex], ctx.discardIndex - ctx.payloadIndex);
  return ctx.interestCount;
}
