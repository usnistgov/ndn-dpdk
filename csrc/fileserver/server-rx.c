#include "server.h"

#include "../core/logger.h"
#include "../ndni/tlv-encoder.h"
#include "fd.h"
#include "naming.h"
#include "op.h"

N_LOG_INIT(FileServer);

static DataEnc_MetaInfoBuffer(15) MetaInfo_Metadata;
static DataEnc_MetaInfoBuffer(15) MetaInfo_Nack;

RTE_INIT(InitMetaInfo)
{
  uint8_t segment0[] = { TtSegmentNameComponent, 1, 0 };
  LName finalBlock0 = (LName){ .length = sizeof(segment0), .value = segment0 };
  DataEnc_PrepareMetaInfo(&MetaInfo_Metadata, ContentBlob, FileServerMetadataFreshness,
                          finalBlock0);
  DataEnc_PrepareMetaInfo(&MetaInfo_Nack, ContentNack, FileServerMetadataFreshness, (LName){ 0 });
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
  RTE_MARKER zeroizeEnd_;
  struct rte_mbuf* interest[MaxBurstSize];
  union
  {
    struct rte_mbuf* payload[MaxBurstSize];
    struct rte_mbuf* discard[2 * MaxBurstSize];
  };
  Packet* data[MaxBurstSize];
} RxBurstCtx;
static_assert(RTE_DIM(((RxBurstCtx*)NULL)->discard) <= UINT8_MAX, "");

__attribute__((nonnull)) static __rte_always_inline bool
FileServerRx_CheckVersion(FileServer* p, FileServerFd* fd, FileServerRequestName rn)
{
  if (likely(rn.version == fd->version)) {
    return true;
  }
  return p->versionBypassHi != 0 && (rn.version >> 32) == p->versionBypassHi;
}

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
  struct io_uring_sqe* sqe = Uring_GetSqe(&p->ur);
  if (unlikely(sqe == NULL)) {
    return FileServerRx_NoSqe(ctx);
  }

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
  ++p->cnt.reqRead;
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
    N_LOGD("Read drop=no-fd");
    goto DROP;
  }
  if (unlikely(fd == FileServer_NotFound)) {
    N_LOGD("Read drop=file-not-found");
    goto DROP;
  }
  if (unlikely(!FileServerFd_IsFile(fd))) {
    N_LOGD("Read fd=%d drop=mode-not-file", fd->fd);
    goto UNREF;
  }
  if (unlikely(!FileServerRx_CheckVersion(p, fd, rn))) {
    N_LOGD("Read fd=%d drop=version-changed rn-version=%" PRIu64 " fd-version=%" PRIu64, fd->fd,
           rn.version, fd->version);
    goto UNREF;
  }
  if (unlikely(rn.segment > fd->lastSeg)) {
    N_LOGD("Read fd=%d drop=segment-out-of-range rn-segment=%" PRIu64 " lastseg=%" PRIu64, fd->fd,
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
  NULLize(fd);
DROP:
  ctx->discard[ctx->discardIndex++] = interest;
}

__attribute__((nonnull)) static __rte_noinline void
FileServerRx_Ls(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn)
{
  ++p->cnt.reqLs;
  struct rte_mbuf* interest = ctx->interest[ctx->interestIndex];
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  LName name = PName_ToLName(&pi->name);

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("Ls drop=no-fd");
    return;
  }
  if (unlikely(fd == FileServer_NotFound)) {
    N_LOGD("Ls drop=not-found");
    return;
  }
  if (unlikely(!FileServerFd_IsDir(fd))) {
    N_LOGD("Ls fd=%d drop=mode-not-dir", fd->fd);
    goto UNREF;
  }
  if (unlikely(!FileServerRx_CheckVersion(p, fd, rn))) {
    N_LOGD("Ls fd=%d drop=version-changed rn-version=%" PRIu64 " fd-version=%" PRIu64, fd->fd,
           rn.version, fd->version);
    goto UNREF;
  }
  if (fd->lsL == UINT32_MAX) {
    bool ok = FileServerFd_GenerateLs(p, fd);
    if (unlikely(!ok)) {
      N_LOGD("Ls fd=%d drop=ls-error", fd->fd);
      goto UNREF;
    }
  }
  if (unlikely(rn.segment > fd->lastSeg)) {
    N_LOGD("Ls fd=%d drop=segment-out-of-range rn-segment=%" PRIu64 " lastseg=%" PRIu64, fd->fd,
           rn.segment, fd->lastSeg);
    goto UNREF;
  }

  uint32_t valueOffset = rn.segment * p->segmentLen;
  uint32_t valueLen = RTE_MIN(p->segmentLen, fd->lsL - valueOffset);
  struct rte_mbuf* payload = ctx->payload[ctx->payloadIndex];
  payload->data_off = p->payloadHeadroom;
  rte_memcpy(rte_pktmbuf_append(payload, valueLen), RTE_PTR_ADD(fd->lsV, valueOffset), valueLen);

  Packet* data = DataEnc_EncodePayload(name, (LName){ 0 }, &fd->meta, payload);
  if (unlikely(data == NULL)) {
    goto ENCERR;
  }
  ++ctx->payloadIndex;

  Mbuf_SetTimestamp(payload, ctx->now);
  *Packet_GetLpL3Hdr(data) = *Packet_GetLpL3Hdr(npkt);
  ctx->data[ctx->dataCount++] = data;
  goto UNREF;

ENCERR:
  N_LOGD("Ls drop=dataenc-error");
  rte_pktmbuf_reset(payload);
UNREF:
  FileServerFd_Unref(p, fd);
  NULLize(fd);
}

__attribute__((nonnull)) static __rte_noinline void
FileServerRx_Metadata(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn)
{
  ++p->cnt.reqMetadata;
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
    metaInfo = &MetaInfo_Nack;
  } else if (unlikely((rn.kind & FileServerRequestLs) != 0 && !FileServerFd_IsDir(fd))) {
    FileServerFd_Unref(p, fd);
    NULLize(fd);
    metaInfo = &MetaInfo_Nack;
  } else {
    bool ok = FileServerFd_EncodeMetadata(p, fd, payload);
    FileServerFd_Unref(p, fd);
    NULLize(fd);
    if (unlikely(!ok)) {
      goto ENCERR;
    }
    metaInfo = &MetaInfo_Metadata;
  }

  struct timespec utcNow;
  if (unlikely(clock_gettime(CLOCK_REALTIME, &utcNow) != 0)) {
    goto ENCERR;
  }
  uint8_t suffixV[20];
  LName suffix = (LName){ .length = 0, .value = suffixV };
  suffix.length =
    Nni_EncodeNameComponent(suffixV, TtVersionNameComponent, FileServerFd_StatTime(utcNow));
  suffix.length +=
    Nni_EncodeNameComponent(RTE_PTR_ADD(suffixV, suffix.length), TtSegmentNameComponent, 0);

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
    case FileServerRequestLs | FileServerRequestVersion | FileServerRequestSegment:
      FileServerRx_Ls(p, ctx, rn);
      break;
    case FileServerRequestMetadata:
    case FileServerRequestLs | FileServerRequestMetadata:
      FileServerRx_Metadata(p, ctx, rn);
      break;
    default:
      N_LOGD("I drop=bad-name rn-kind=%" PRIx32, (uint32_t)rn.kind);
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
    // upon failure, callee sets ctx.interestIndex to ctx.interestCount, stopping the loop
  }
  if (FileServer_EnableIovBatching && likely(ctx.op != NULL)) {
    FileServerRx_SubmitReadv(p, &ctx);
  }

  Uring_Submit(&p->ur, p->uringWaitLbound, MaxBurstSize);
  Face_TxBurst(p->face, ctx.data, ctx.dataCount);
  rte_pktmbuf_free_bulk(&ctx.discard[ctx.payloadIndex], ctx.discardIndex - ctx.payloadIndex);
  return ctx.interestCount;
}
