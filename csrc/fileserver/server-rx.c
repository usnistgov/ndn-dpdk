#include "server.h"

#include "../core/logger.h"
#include "../ndni/tlv-encoder.h"
#include "fd.h"
#include "naming.h"
#include "op.h"

N_LOG_INIT(FileServer);

DataEnc_MetaInfoBuffer(15) MetadataMetaInfo;
DataEnc_MetaInfoBuffer(15) NackMetaInfo;

RTE_INIT(InitMetaInfo)
{
  uint8_t segment0[10];
  segment0[0] = TtSegmentNameComponent;
  segment0[1] = Nni_Encode(&segment0[2], 0);
  LName finalBlock = (LName){ .length = 2 + segment0[1], .value = segment0 };
  DataEnc_MustPrepareMetaInfo(&MetadataMetaInfo, ContentBlob, 1, finalBlock);

  DataEnc_MustPrepareMetaInfo(&NackMetaInfo, ContentNack, 1, (LName){ 0 });
}

typedef struct RxBurstCtx
{
  TscTime now;
  FileServerOp* op;
  uint8_t interestIndex; ///< interest[:interestIndex] are processed
  uint8_t interestCount; ///< interest[interestIndex:interestCount] are unprocessed
  uint8_t payloadIndex;  ///< payload[payloadIndex:] are unused
  uint8_t discardIndex;  ///< discard[MaxBurstSize:discardIndex] are dropped Interests
  uint8_t dataCount;     ///< data[:dataCount] are Data packets to be sent
  bool hasSqe;
  char zeroizeEnd_[0];
  struct rte_mbuf* interest[MaxBurstSize];
  union
  {
    struct rte_mbuf* payload[MaxBurstSize];
    struct rte_mbuf* discard[2 * MaxBurstSize];
  };
  Packet* data[MaxBurstSize];
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
FileServerRx_SubmitReadv(FileServer* p, RxBurstCtx* ctx)
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
FileServerRx_Read(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn)
{
  struct rte_mbuf* interest = ctx->interest[ctx->interestIndex];
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);

  LName prefix = FileServer_GetPrefix(&pi->name);
  struct rte_mbuf* payload = ctx->payload[ctx->payloadIndex];
  payload->data_off = p->payloadHeadroom;

  if (FileServer_EnableIovBatching && likely(ctx->op != NULL)) {
    if (likely(FileServerOp_Follows(ctx->op, prefix, rn.segment))) {
      goto ADD_IOV;
    }
    if (unlikely(!FileServerRx_SubmitReadv(p, ctx))) {
      // not `goto DROP` because FileServerRx_NoSqe has discarded 'interest',
      // which was "unprocessed" before returning to RxBurst
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
    goto DROP;
  }
  if (unlikely(!FileServerFd_IsFile(fd))) {
    N_LOGD("I drop=mode-not-file");
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
  FileServerOp_AppendIov(ctx->op, payload, p->segmentLen, interest);
  ++ctx->payloadIndex;

  if (!FileServer_EnableIovBatching) {
    FileServerRx_SubmitReadv(p, ctx);
  }
  return;
UNREF:
  FileServerFd_Unref(p, fd);
DROP:
  ctx->discard[ctx->discardIndex++] = interest;
}

__attribute__((nonnull)) static __rte_noinline void
FileServerRx_Metadata(FileServer* p, RxBurstCtx* ctx)
{
  struct rte_mbuf* interest = ctx->interest[ctx->interestIndex];
  ctx->discard[ctx->discardIndex++] = interest;
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  LName name = PName_ToLName(&pi->name);

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("Metadata drop=no-fd");
    return;
  }

  struct rte_mbuf* payload = ctx->payload[ctx->payloadIndex];
  payload->data_off = p->payloadHeadroom;
  const void* metaInfo = NULL;

  if (unlikely(fd == FileServer_NotFound)) {
    metaInfo = &NackMetaInfo;
  } else {
    metaInfo = &MetadataMetaInfo;
    bool ok = FileServerFd_EncodeMetadata(p, fd, payload);
    FileServerFd_Unref(p, fd);
    if (unlikely(!ok)) {
      goto ENCERR;
    }
  }

  uint8_t suffixV[20];
  suffixV[0] = TtVersionNameComponent;
  suffixV[1] = Nni_Encode(&suffixV[2], ctx->now);
  LName suffix = (LName){ .length = 2 + suffixV[1], .value = suffixV };
  *(unaligned_uint16_t*)RTE_PTR_ADD(suffixV, suffix.length) =
    TlvEncoder_ConstTL1(TtSegmentNameComponent, 1);
  suffixV[suffix.length + 2] = 0;
  suffix.length += 3;
  Packet* data = DataEnc_EncodePayload(name, suffix, metaInfo, payload);
  if (unlikely(data == NULL)) {
    goto ENCERR;
  }
  ++ctx->payloadIndex;

  Mbuf_SetTimestamp(payload, ctx->now);
  *Packet_GetLpL3Hdr(data) = *Packet_GetLpL3Hdr(npkt);
  ctx->data[ctx->dataCount++] = data;
  return;

ENCERR:
  N_LOGD("Metadata drop=dataenc-error");
  rte_pktmbuf_reset(payload);
}

__attribute__((nonnull)) static inline void
FileServerRx_ProcessInterest(FileServer* p, RxBurstCtx* ctx)
{
  struct rte_mbuf* interest = ctx->interest[ctx->interestIndex];
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  FileServerRequestName rn = FileServer_ParseRequest(pi);
  switch ((uint32_t)rn.kind) {
    case FileServerRequestVersion | FileServerRequestSegment:
      FileServerRx_Read(p, ctx, rn);
      break;
    case FileServerRequestMetadata:
      FileServerRx_Metadata(p, ctx);
      break;
    default:
      N_LOGD("I drop=bad-name rn.kind=%" PRIx32, (uint32_t)rn.kind);
      ctx->discard[ctx->discardIndex++] = interest;
      break;
  }
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
  if (FileServer_EnableIovBatching && likely(ctx.op != NULL)) {
    FileServerRx_SubmitReadv(p, &ctx);
  }

  if (likely(ctx.hasSqe)) {
    res = io_uring_submit(&p->uring);
    if (unlikely(res < 0)) {
      N_LOGE("io_uring_submit errno=%d", -res);
    }
  }

  Face_TxBurst(p->face, ctx.data, ctx.dataCount);
  rte_pktmbuf_free_bulk(&ctx.discard[ctx.payloadIndex], ctx.discardIndex - ctx.payloadIndex);
  return ctx.interestCount;
}
