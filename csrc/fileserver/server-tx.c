#include "server.h"

#include "../core/logger.h"
#include "fd.h"
#include "op.h"

N_LOG_INIT(FileServer);

typedef struct TxBurstCtx {
  TscTime now;
  uint16_t index;    ///< cqe[:index] are processed
  uint16_t nData;    ///< data[:nData] are to be transmitted
  uint16_t nDiscard; ///< discard[:nDiscard] are Data to be freed
  uint8_t congMark;
  struct io_uring_cqe* cqe[MaxBurstSize];
  Packet* data[MaxBurstSize];
  struct rte_mbuf* discard[MaxBurstSize];
} TxBurstCtx;
static_assert(RTE_DIM(((TxBurstCtx*)NULL)->discard) <= UINT16_MAX, "");

__attribute__((nonnull)) static inline void
FileServerTx_ProcessCqe(FileServer* p, TxBurstCtx* ctx) {
  struct io_uring_cqe* cqe = ctx->cqe[ctx->index];
  FileServerOp* op = io_uring_cqe_get_data(cqe);
  FileServerFd* fd = op->fd;

  if (unlikely(cqe->res < 0)) {
    N_LOGD("CQE fd=%d iovcnt=%d drop=cqe-error" N_LOG_ERROR_ERRNO, fd->fd, op->iovcnt, cqe->res);
    goto FREE_DATA;
  }
  if (unlikely((uint16_t)cqe->res != op->contentLen)) {
    N_LOGD("CQE fd=%d iovcnt=%d drop=short-read cqe-res=%" PRId32
           " content-len=%" PRIu16 N_LOG_ERROR_BLANK,
           fd->fd, op->iovcnt, (int32_t)cqe->res, op->contentLen);
    goto FREE_DATA;
  }

  N_LOGV("CQE fd=%d iovcnt=%d res=%" PRId32, fd->fd, op->iovcnt, (int32_t)cqe->res);
  Packet* data = FileServer_SignAndSend(p, ctx, fd, "CQE", op->data, op->interestL3);
  if (likely(data != NULL)) {
    LpL3* dataL3 = Packet_GetLpL3Hdr(data);
    dataL3->congMark = RTE_MAX(dataL3->congMark, ctx->congMark);
    ctx->congMark = 0;
  }
  goto FREE_OP;

FREE_DATA:
  rte_pktmbuf_free(op->data);
FREE_OP:
  rte_mempool_put(p->opMp, op);
  FileServerFd_Unref(p, fd);
  NULLize(fd);
}

uint32_t
FileServer_TxBurst(FileServer* p) {
  TxBurstCtx ctx;
  ctx.now = rte_get_tsc_cycles();
  ctx.congMark = (uint8_t)(p->ur.nPending >= p->uringCongestionLbound);
  ctx.nData = 0;
  ctx.nDiscard = 0;

  uint32_t nCqe = Uring_PeekCqes(&p->ur, ctx.cqe, RTE_DIM(ctx.cqe));
  for (ctx.index = 0; ctx.index < nCqe; ++ctx.index) {
    FileServerTx_ProcessCqe(p, &ctx);
  }
  Uring_SeenCqes(&p->ur, nCqe);

  Face_TxBurst(p->face, ctx.data, ctx.nData);
  rte_pktmbuf_free_bulk(ctx.discard, ctx.nDiscard);
  return nCqe;
}
